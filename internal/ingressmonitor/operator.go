package ingressmonitor

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned"
	crdscheme "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/scheme"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/informers/externalversions"

	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/apis/extensions"

	"github.com/dchest/blake2b"
)

const (
	monitorLabel     = "ingressmonitor.sphc.io/monitor"
	ingressLabel     = "ingressmonitor.sphc.io/ingress"
	ingressHostLabel = "ingressmonitor.sphc.io/ingress-path"
)

var encoder = base32.HexEncoding.WithPadding(base32.NoPadding)

// Operator is the operator that handles configuring the Monitors.
type Operator struct {
	kubeClient kubernetes.Interface
	imClient   versioned.Interface

	imInformer      externalversions.SharedInformerFactory
	providerFactory provider.FactoryInterface

	monitorQueue        workqueue.RateLimitingInterface
	ingressMonitorQueue workqueue.RateLimitingInterface
}

// NewOperator sets up a new IngressMonitor Operator which will watch for
// providers and monitors.
func NewOperator(
	kc kubernetes.Interface, imc versioned.Interface,
	namespace string, resync time.Duration,
	providerFactory provider.FactoryInterface) (*Operator, error) {

	// Register the scheme with the client so we can use it through the API
	crdscheme.AddToScheme(scheme.Scheme)

	op := &Operator{
		kubeClient:          kc,
		imClient:            imc,
		imInformer:          externalversions.NewSharedInformerFactory(imc, resync),
		providerFactory:     providerFactory,
		monitorQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Monitors"),
		ingressMonitorQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "IngressMonitors"),
	}

	// Add EventHandlers for all objects we want to track
	op.imInformer.Ingressmonitor().V1alpha1().Monitors().Informer().AddEventHandler(op)
	op.imInformer.Ingressmonitor().V1alpha1().IngressMonitors().Informer().AddEventHandler(op)

	return op, nil
}

// Run starts the Operator and blocks until a message is received on stopCh.
func (o *Operator) Run(stopCh <-chan struct{}) error {
	defer o.monitorQueue.ShutDown()
	defer o.ingressMonitorQueue.ShutDown()

	log.Printf("Starting IngressMonitor Operator")

	log.Printf("Starting the informers")
	o.imInformer.Start(stopCh)

	log.Printf("Starting the workers")
	for i := 0; i < 4; i++ {
		go wait.Until(runWorker(o.processNextIngressMonitor), time.Second, stopCh)
	}

	<-stopCh
	log.Printf("Stopping IngressMonitor Operator")

	return nil
}

func runWorker(queue func() bool) func() {
	return func() {
		for queue() {
		}
	}
}

func (o *Operator) processNextIngressMonitor() bool {
	return o.handleNextItem("IngressMonitors", o.ingressMonitorQueue, o.handleIngressMonitor)
}

func (o *Operator) handleNextItem(name string, queue workqueue.RateLimitingInterface, handlerFunc func(string) error) bool {
	obj, shutdown := queue.Get()

	if shutdown {
		return false
	}

	// wrap this in a function so we can use defer to mark processing the item
	// as done.
	err := func(obj interface{}) error {
		defer queue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			queue.Forget(obj)

			log.Printf("Expected object name in %s workqueue, got %#v", name, obj)
			return nil
		}

		if err := handlerFunc(key); err != nil {
			return fmt.Errorf("Error handling '%s' in %s workqueue: %s", key, name, err)
		}

		queue.Forget(obj)
		log.Printf("Processed '%s' in %s workqueue", key, name)
		return nil
	}(obj)

	if err != nil {
		log.Printf(err.Error())
		return false
	}

	return true
}

func (o *Operator) enqueueItem(queue workqueue.RateLimitingInterface, obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	queue.AddRateLimited(key)
}

func (o *Operator) enqueueIngressMonitor(im *v1alpha1.IngressMonitor) {
	o.enqueueItem(o.ingressMonitorQueue, im)
}

// OnAdd handles adding of IngressMonitors and Ingresses and sets up the
// appropriate monitor with the configured providers.
func (o *Operator) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.IngressMonitor:
		o.enqueueIngressMonitor(obj)
	case *v1alpha1.Monitor:
		if err := o.handleMonitor(obj); err != nil {
			log.Printf("Error adding Monitor %s:%s: %s", obj.Namespace, obj.Name, err)
		}
	}
}

// OnUpdate handles updates of IngressMonitors anad Ingresses and configures the
// checks with the configured providers.
func (o *Operator) OnUpdate(old, new interface{}) {
	switch obj := new.(type) {
	case *v1alpha1.IngressMonitor:
		o.enqueueIngressMonitor(obj)
	case *v1alpha1.Monitor:
		// GC old objects, we do this on every run - even resyncs - so we can be
		// sure that even when an Ingress changes it's spec, we update our
		// configuration as well.
		if err := o.garbageCollectMonitors(obj); err != nil {
			log.Printf("Error doing garbage collection for %s:%s: %s", obj.Namespace, obj.Name, err)
		}

		if err := o.handleMonitor(obj); err != nil {
			log.Printf("Error updating Monitor %s:%s: %s", obj.Namespace, obj.Name, err)
		}
	}
}

// OnDelete handles deletion of IngressMonitors and Ingresses and deletes
// monitors from the configured providers.
func (o *Operator) OnDelete(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.IngressMonitor:
		cl, err := o.providerFactory.From(obj.Spec.Provider)
		if err != nil {
			log.Printf("Could not get provider for IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		if err := cl.Delete(obj.Status.ID); err != nil {
			log.Printf("Could not delete IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}
	case *v1alpha1.Monitor:
		imList, err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).
			List(listOptions(map[string]string{monitorLabel: obj.Name}))
		if err != nil {
			log.Printf("Could not list IngressMonitors for Monitors %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		for _, im := range imList.Items {
			if err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).
				Delete(im.Name, &metav1.DeleteOptions{}); err != nil {
				log.Printf("Could not delete IngressMonitor %s for Monitors %s:%s: %s", im.Name, obj.Namespace, obj.Name, err)
			}
		}
	}
}

// handleIngressMonitor handles IngressMonitors in a way that it knows how to
// deal with creating and updating resources.
func (o *Operator) handleIngressMonitor(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("Invalid Resource Key for IngressMonitor: %s", key)
	}

	obj, err := o.imClient.Ingressmonitor().IngressMonitors(namespace).
		Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// item might have been deleted by now, ignore it
			return nil
		}

		return fmt.Errorf("Error fetching the IngressMonitor for %s: %s", key, err)
	}

	cl, err := o.providerFactory.From(obj.Spec.Provider)
	if err != nil {
		return fmt.Errorf("Error fetching provider '%s': %s", obj.Spec.Provider.Type, err)
	}

	var id string
	if obj.Status.ID != "" {
		// This object hasn't been created yet, do so!
		id, err = cl.Update(obj.Status.ID, obj.Spec.Template)
	} else {
		id, err = cl.Create(obj.Spec.Template)
	}

	if err != nil {
		return err
	}

	// The ID has changed, update the status. This could happen when the test
	// has been removed from the provider. The operator ensures that the test
	// will be present, and thus create a new one.
	if obj.Status.ID != id {
		obj.Status.ID = id
		_, err = o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).Update(obj)
	}

	return err
}

// garbgageCollectMonitors finds all IngressMonitors that are linked to a
// specific Monitor which shouldn't be configured in the cluster anymore.
// It does this by fetching all Ingresses which should currently be set up for
// the monitor and then fetching the IngressMonitors which are linked to the
// specified Monitor.
// If one of the monitors isn't linked to the Ingress, it gets marked for
// deletion.
func (o *Operator) garbageCollectMonitors(obj *v1alpha1.Monitor) error {
	ingressList, err := o.kubeClient.Extensions().Ingresses(obj.Namespace).
		List(listOptions(obj.Spec.Selector.MatchLabels))
	if err != nil {
		return err
	}

	// We'll calculate all the IngressMonitors that shouldn't be tracked
	// anymore and delete them. We can do this by fetching all
	// IngressMonitors where the owner is this Monitor, go over them all and
	// see if there are any where the Ingress Owner isn't in the new Ingress
	// List.
	imList, err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).
		List(listOptions(map[string]string{monitorLabel: obj.Name}))
	if err != nil {
		return err
	}
	for _, im := range imList.Items {
		var isActive bool

		// Go through all newly selected Ingresses and see if this
		// IngressMonitor is active for any of them. We do this by first
		// validating if it's controlled by the Ingress, and if so we check if
		// it matches any of the rules. Ingresses might change which means a
		// specific rule can be dropped. We need to GC that.
		for _, ing := range ingressList.Items {
			if metav1.IsControlledBy(&im, &ing) {
				for _, rule := range ing.Spec.Rules {
					if rule.Host == im.Labels[ingressHostLabel] {
						isActive = true
					}
				}
			}
		}

		// The IngressMonitor doesn't appear in any newly selected Ingress
		// anymore, which means it's ready for GarbageCollection. Delete the
		// IngressMonitor Resource from the server, which will then trigger a
		// reconciliation to take care of actually removing the monitor with the
		// provider.
		if !isActive {
			if err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).
				Delete(im.Name, &metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *Operator) handleMonitor(iObj interface{}) error {
	obj := iObj.(*v1alpha1.Monitor)
	ingressList, err := o.kubeClient.Extensions().Ingresses(obj.Namespace).
		List(listOptions(obj.Spec.Selector.MatchLabels))
	if err != nil {
		return fmt.Errorf("Could not list Ingresses: %s", err)
	}

	// fetch the referenced provider
	prov, err := o.imClient.Ingressmonitor().Providers(obj.Namespace).
		Get(obj.Spec.Provider.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Could not get Provider: %s", err)
	}

	// fetch the referenced template
	tmpl, err := o.imClient.Ingressmonitor().MonitorTemplates().
		Get(obj.Spec.Template.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Could not get MonitorTemplate: %s", err)
	}

	// reconcile the newly selected Ingresses. We'll create new IngressMonitors
	// for each Ingress and it's subsequent rules. If it already exists, we
	// update it.
	for _, ing := range ingressList.Items {
		for _, rule := range ing.Spec.Rules {
			name := fmt.Sprintf("%s-%s", ing.Name, shortHash(rule.Host, 16))

			// we can only assign one reference that controls the object, ensure
			// that it's the Ingress so that we can still perform garbage
			// collection.
			monitorReference := *metav1.NewControllerRef(
				obj,
				v1alpha1.SchemeGroupVersion.WithKind("Monitor"),
			)
			monitorReference.Controller = nil

			templateSpec := tmpl.Spec
			tplName, err := templatedName(ing, templateSpec)
			if err != nil {
				return fmt.Errorf("Could not get templated name: %s", err)
			}
			templateSpec.Name = tplName

			healthPath := "/_healthz"

			scheme := "http://"
		TLSLoop:
			for _, tlsList := range ing.Spec.TLS {
				for _, host := range tlsList.Hosts {
					if host == rule.Host {
						scheme = "https://"
						break TLSLoop
					}
				}
			}

			if templateSpec.HTTP.Endpoint != nil {
				healthPath = *templateSpec.HTTP.Endpoint
			}
			monitorURL := fmt.Sprintf("%s%s%s", scheme, rule.Host, healthPath)
			templateSpec.HTTP.URL = monitorURL

			im := &v1alpha1.IngressMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ing.Namespace,
					// Add OwnerReferences to the IngressMonitor so we can
					// automatically Garbage Collect when either a Monitor is
					// removed or when the Ingress is removed. This way we don't
					// have to set this up ourselves.
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(
							&ing,
							extensions.SchemeGroupVersion.WithKind("Ingress"),
						),
						monitorReference,
					},
					// Set some labels so it's easier to filter later on
					Labels: map[string]string{
						monitorLabel:     obj.Name,
						ingressLabel:     ing.Name,
						ingressHostLabel: rule.Host,
					},
				},
				Spec: v1alpha1.IngressMonitorSpec{
					Provider: v1alpha1.NamespacedProvider{
						Namespace:    obj.Namespace,
						ProviderSpec: prov.Spec,
					},
					Template: templateSpec,
				},
			}

			gIM, err := o.imClient.Ingressmonitor().IngressMonitors(im.Namespace).
				Get(im.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				_, err = o.imClient.Ingressmonitor().
					IngressMonitors(im.Namespace).Create(im)
			} else if err == nil {
				im.ObjectMeta = gIM.ObjectMeta
				im.TypeMeta = gIM.TypeMeta
				im.Status = gIM.Status
				im.Status.IngressName = ing.Name

				_, err = o.imClient.Ingressmonitor().
					IngressMonitors(im.Namespace).Update(im)
			}

			if err != nil {
				return fmt.Errorf("Could not ensure IngressMonitor: %s", err)
			}
		}
	}

	return nil
}

func listOptions(lbls map[string]string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: labels.FormatLabels(lbls),
	}
}

// shortHash creates a shortened hash from the given string. The hash is
// lowercase base32 encoded, suitable for DNS use, and at most "len" characters
// long.
func shortHash(data string, len int) string {
	b2b, _ := blake2b.New(&blake2b.Config{Size: uint8(len * 5 / 8)})
	b2b.Write([]byte(data))
	return strings.ToLower(encoder.EncodeToString(b2b.Sum(nil)))
}

func templatedName(ing v1beta1.Ingress, sp v1alpha1.MonitorTemplateSpec) (string, error) {
	tpl, err := template.New("im-name").Parse(sp.Name)
	if err != nil {
		return "", err
	}

	data := struct {
		IngressName      string
		IngressNamespace string
	}{
		IngressName:      ing.Name,
		IngressNamespace: ing.Namespace,
	}

	buf := bytes.NewBufferString("")
	if err := tpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
