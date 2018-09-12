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
	tv1alpha1 "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/typed/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/informers/externalversions"
	lv1alpha1 "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/listers/ingressmonitor/v1alpha1"

	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ev1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
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
	imClient   tv1alpha1.IngressmonitorV1alpha1Interface

	providerFactory provider.FactoryInterface

	imInformer   cache.SharedIndexInformer
	mInformer    cache.SharedIndexInformer
	ingInformer  cache.SharedIndexInformer
	provInformer cache.SharedIndexInformer
	mtInformer   cache.SharedIndexInformer

	informers []namedInformer

	ingLister  ev1beta1.IngressLister
	provLister lv1alpha1.ProviderLister
	mtLister   lv1alpha1.MonitorTemplateLister

	monitorQueue        workqueue.RateLimitingInterface
	ingressMonitorQueue workqueue.RateLimitingInterface
}

type namedInformer struct {
	name     string
	informer cache.SharedIndexInformer
}

// NewOperator sets up a new IngressMonitor Operator which will watch for
// providers and monitors.
func NewOperator(
	kc kubernetes.Interface, imc versioned.Interface,
	namespace string, resync time.Duration,
	providerFactory provider.FactoryInterface) (*Operator, error) {

	// Register the scheme with the client so we can use it through the API
	crdscheme.AddToScheme(scheme.Scheme)

	imInformer := externalversions.NewSharedInformerFactory(imc, resync).Ingressmonitor().V1alpha1()
	k8sInformer := informers.NewSharedInformerFactory(kc, resync)

	op := &Operator{
		kubeClient:          kc,
		imClient:            imc.Ingressmonitor(),
		providerFactory:     providerFactory,
		monitorQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Monitors"),
		ingressMonitorQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "IngressMonitors"),

		imInformer:   imInformer.IngressMonitors().Informer(),
		mInformer:    imInformer.Monitors().Informer(),
		provInformer: imInformer.Providers().Informer(),
		mtInformer:   imInformer.MonitorTemplates().Informer(),

		ingInformer: k8sInformer.Extensions().V1beta1().Ingresses().Informer(),
	}

	// Add EventHandlers for all objects we want to track
	op.imInformer.AddEventHandler(op)
	op.mInformer.AddEventHandler(op)

	// set up listers
	op.ingLister = ev1beta1.NewIngressLister(op.ingInformer.GetIndexer())
	op.provLister = lv1alpha1.NewProviderLister(op.provInformer.GetIndexer())
	op.mtLister = lv1alpha1.NewMonitorTemplateLister(op.mtInformer.GetIndexer())

	op.informers = []namedInformer{
		{"IngressMonitor", op.imInformer},
		{"Monitor", op.mInformer},
		{"Ingress", op.ingInformer},
		{"Provider", op.provInformer},
		{"MonitorTemplate", op.mtInformer},
	}

	return op, nil
}

// Run starts the Operator and blocks until a message is received on stopCh.
func (o *Operator) Run(stopCh <-chan struct{}) error {
	defer o.monitorQueue.ShutDown()
	defer o.ingressMonitorQueue.ShutDown()

	log.Printf("Starting IngressMonitor Operator")
	if err := o.connectToCluster(stopCh); err != nil {
		return err
	}

	log.Printf("Starting the informers")
	if err := o.startInformers(stopCh); err != nil {
		return err
	}

	log.Printf("Starting the workers")
	for i := 0; i < 4; i++ {
		go wait.Until(runWorker(o.processNextIngressMonitor), time.Second, stopCh)
		go wait.Until(runWorker(o.processNextMonitor), time.Second, stopCh)
	}

	<-stopCh
	log.Printf("Stopping IngressMonitor Operator")

	return nil
}

func (o *Operator) connectToCluster(stopCh <-chan struct{}) error {
	errCh := make(chan error)
	go func() {
		v, err := o.kubeClient.Discovery().ServerVersion()
		if err != nil {
			errCh <- fmt.Errorf("Could not communicate with the server: %s", err)
			return
		}

		log.Printf("Connected to the cluster (version %s)", v)
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-stopCh:
		return nil
	}

	return nil
}

func (o *Operator) startInformers(stopCh <-chan struct{}) error {
	for _, inf := range o.informers {
		log.Printf("Starting informer %s", inf.name)
		go inf.informer.Run(stopCh)
	}

	if err := o.waitForCaches(stopCh); err != nil {
		return err
	}

	log.Printf("Synced all caches")
	return nil
}

func (o *Operator) waitForCaches(stopCh <-chan struct{}) error {
	var syncFailed bool
	for _, inf := range o.informers {
		log.Printf("Waiting for cache sync for %s", inf.name)
		if !cache.WaitForCacheSync(stopCh, inf.informer.HasSynced) {
			log.Printf("Could not sync cache for %s", inf.name)
			syncFailed = true
		} else {
			log.Printf("Synced cache for %s", inf.name)
		}
	}

	if syncFailed {
		return fmt.Errorf("could not sync cache")
	}

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

func (o *Operator) processNextMonitor() bool {
	return o.handleNextItem("Monitors", o.monitorQueue, o.handleMonitor)
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
		log.Printf("Synced '%s' in %s workqueue", key, name)
		return nil
	}(obj)

	if err != nil {
		log.Printf(err.Error())
	}

	return true
}

func (o *Operator) enqueueItem(queue workqueue.RateLimitingInterface, obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		return
	}

	queue.AddRateLimited(key)
}

func (o *Operator) enqueueIngressMonitor(im *v1alpha1.IngressMonitor) {
	o.enqueueItem(o.ingressMonitorQueue, im)
}

func (o *Operator) enqueueMonitor(m *v1alpha1.Monitor) {
	o.enqueueItem(o.monitorQueue, m)
}

// OnAdd handles adding of IngressMonitors and Ingresses and sets up the
// appropriate monitor with the configured providers.
func (o *Operator) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.IngressMonitor:
		o.enqueueIngressMonitor(obj)
	case *v1alpha1.Monitor:
		o.enqueueMonitor(obj)
	}
}

// OnUpdate handles updates of IngressMonitors anad Ingresses and configures the
// checks with the configured providers.
func (o *Operator) OnUpdate(old, new interface{}) {
	switch obj := new.(type) {
	case *v1alpha1.IngressMonitor:
		o.enqueueIngressMonitor(obj)
	case *v1alpha1.Monitor:
		o.enqueueMonitor(obj)
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
		imList, err := o.imClient.IngressMonitors(obj.Namespace).
			List(listOptions(map[string]string{monitorLabel: obj.Name}))
		if err != nil {
			log.Printf("Could not list IngressMonitors for Monitors %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		for _, im := range imList.Items {
			log.Printf("Deleting IngressMonitor `%s:%s` associated with deleted Monitor `%s:%s`", im.Namespace, im.Name, obj.Namespace, obj.Name)
			if err := o.imClient.IngressMonitors(obj.Namespace).
				Delete(im.Name, &metav1.DeleteOptions{}); err != nil {
				log.Printf("Could not delete IngressMonitor %s for Monitors %s:%s: %s", im.Name, obj.Namespace, obj.Name, err)
			}
		}
	}
}

// handleIngressMonitor handles IngressMonitors in a way that it knows how to
// deal with creating and updating resources.
func (o *Operator) handleIngressMonitor(key string) error {
	item, exists, err := o.imInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	// it's been deleted before we start handling it
	if !exists {
		return nil
	}

	obj := item.(*v1alpha1.IngressMonitor)

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
		_, err = o.imClient.IngressMonitors(obj.Namespace).Update(obj)
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
	ingLabels, err := metav1.LabelSelectorAsSelector(obj.Spec.Selector)
	if err != nil {
		return fmt.Errorf("Could not create label selector for %s:%s: %s", obj.Namespace, obj.Name, err)
	}

	ingressList, err := o.ingLister.Ingresses(obj.Namespace).List(ingLabels)
	if err != nil {
		return fmt.Errorf("Could not list Ingresses: %s", err)
	}

	// We'll calculate all the IngressMonitors that shouldn't be tracked
	// anymore and delete them. We can do this by fetching all
	// IngressMonitors where the owner is this Monitor, go over them all and
	// see if there are any where the Ingress Owner isn't in the new Ingress
	// List.
	imLabels := labels.SelectorFromSet(map[string]string{monitorLabel: obj.Name})
	cache.ListAllByNamespace(o.imInformer.GetIndexer(), obj.Namespace, imLabels, func(imObj interface{}) {
		im := imObj.(*v1alpha1.IngressMonitor)
		var isActive bool

		// Go through all newly selected Ingresses and see if this
		// IngressMonitor is active for any of them. We do this by first
		// validating if it's controlled by the Ingress, and if so we check if
		// it matches any of the rules. Ingresses might change which means a
		// specific rule can be dropped. We need to GC that.
		for _, ing := range ingressList {
			if metav1.IsControlledBy(im, ing) {
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
			log.Printf("Deleting IngressMonitor %s:%s with GC", im.Namespace, im.Name)
			if err := o.imClient.IngressMonitors(im.Namespace).
				Delete(im.Name, &metav1.DeleteOptions{}); err != nil {

				log.Printf("Could not delete IngressMonitor %s:%s: %s", im.Namespace, im.Name, err)
			}
		}
	})

	return nil
}

func (o *Operator) handleMonitor(key string) error {
	item, exists, err := o.mInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	// it's been deleted before we start handling it
	if !exists {
		return nil
	}

	obj := item.(*v1alpha1.Monitor)
	if err := o.garbageCollectMonitors(obj); err != nil {
		return fmt.Errorf("Error doing garbage collection for %s:%s: %s", obj.Namespace, obj.Name, err)
	}

	ingLabels, err := metav1.LabelSelectorAsSelector(obj.Spec.Selector)
	if err != nil {
		return fmt.Errorf("Could not create label selector for %s:%s: %s", obj.Namespace, obj.Name, err)
	}

	ingressList, err := o.ingLister.Ingresses(obj.Namespace).List(ingLabels)
	if err != nil {
		return fmt.Errorf("Could not list Ingresses: %s", err)
	}

	if len(ingressList) == 0 {
		log.Printf("No ingresses selected for %s:%s", obj.Namespace, obj.Name)
		return nil
	}

	prov, err := o.provLister.Providers(obj.Namespace).Get(obj.Spec.Provider.Name)
	if err != nil {
		return fmt.Errorf("Could not get Provider %s:%s: %s", obj.Namespace, obj.Spec.Provider.Name, err)
	}

	tmpl, err := o.mtLister.MonitorTemplates(obj.Namespace).Get(obj.Spec.Template.Name)
	if err != nil {
		return fmt.Errorf("Could not get MonitorTemplate %s: %s", obj.Spec.Template.Name, err)
	}

	// reconcile the newly selected Ingresses. We'll create new IngressMonitors
	// for each Ingress and it's subsequent rules. If it already exists, we
	// update it.
	for _, ing := range ingressList {
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
							ing,
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

			gIM, err := o.imClient.IngressMonitors(im.Namespace).
				Get(im.Name, metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				_, err = o.imClient.IngressMonitors(im.Namespace).Create(im)
			} else if err == nil {
				im.ObjectMeta = gIM.ObjectMeta
				im.TypeMeta = gIM.TypeMeta
				im.Status = gIM.Status
				im.Status.IngressName = ing.Name

				_, err = o.imClient.IngressMonitors(im.Namespace).Update(im)
			}

			if err != nil {
				return fmt.Errorf("Could not ensure IngressMonitor: %s", err)
			}

			log.Printf("Successfully synced IngressMonitor %s:%s", im.Namespace, im.Name)
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

func templatedName(ing *v1beta1.Ingress, sp v1alpha1.MonitorTemplateSpec) (string, error) {
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
