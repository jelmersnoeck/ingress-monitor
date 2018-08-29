package ingressmonitor

import (
	"log"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned"
	crdscheme "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/scheme"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/informers/externalversions"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

const (
	monitorLabel     = "ingressmonitor.sphc.io/monitor"
	ingressLabel     = "ingressmonitor.sphc.io/ingress"
	ingressHostLabel = "ingressmonitor.sphc.io/ingress-path"
)

// Operator is the operator that handles configuring the Monitors.
type Operator struct {
	kubeClient kubernetes.Interface
	imClient   versioned.Interface

	imInformer  externalversions.SharedInformerFactory
	ingInformer informers.SharedInformerFactory
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
		kubeClient: kc,
		imClient:   imc,

		imInformer:  externalversions.NewSharedInformerFactory(imc, resync),
		ingInformer: informers.NewSharedInformerFactory(kc, resync),
	}

	// Add EventHandlers for all objects we want to track
	op.imInformer.Ingressmonitor().V1alpha1().Monitors().Informer().AddEventHandler(op)
	op.imInformer.Ingressmonitor().V1alpha1().Providers().Informer().AddEventHandler(op)
	op.ingInformer.Extensions().V1beta1().Ingresses().Informer().AddEventHandler(op)

	return op, nil
}

// Run starts the Operator and blocks until a message is received on stopCh.
func (o *Operator) Run(stopCh <-chan struct{}) error {
	log.Printf("Starting IngressMonitor Operator")

	log.Printf("Starting the informers")
	o.imInformer.Start(stopCh)
	o.ingInformer.Start(stopCh)

	<-stopCh
	log.Printf("Stopping IngressMonitor Operator")

	return nil
}

// OnAdd handles adding of IngressMonitors and Ingresses and sets up the
// appropriate monitor with the configured providers.
func (o *Operator) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.IngressMonitor:
		// This object has an ID so it's already been configured. We could be in
		// a restart cycle. The OnUpdate handler will take care in a resync to
		// update the checks to the latest.
		if obj.Status.ID != "" {
			return
		}

		cl, err := provider.From(obj.Spec.Provider)
		if err != nil {
			log.Printf("Could not get provider for IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		id, err := cl.Create(obj.Spec.Template)
		if err != nil {
			log.Printf("Could not create IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		obj.Status.ID = id
		if _, err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).Update(obj); err != nil {
			log.Printf("Could not update IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}
	case *v1alpha1.Monitor:
		if err := o.handleMonitor(obj); err != nil {
			log.Printf("Error adding monitor to the cluster: %s", err)
		}
	}
}

// OnUpdate handles updates of IngressMonitors anad Ingresses and configures the
// checks with the configured providers.
func (o *Operator) OnUpdate(old, new interface{}) {
	switch obj := new.(type) {
	case *v1alpha1.IngressMonitor:
		cl, err := provider.From(obj.Spec.Provider)
		if err != nil {
			log.Printf("Could not get provider for IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		if err := cl.Update(obj.Status.ID, obj.Spec.Template); err != nil {
			log.Printf("Could not update IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}
	}
}

// OnDelete handles deletion of IngressMonitors and Ingresses and deletes
// monitors from the configured providers.
func (o *Operator) OnDelete(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.IngressMonitor:
		cl, err := provider.From(obj.Spec.Provider)
		if err != nil {
			log.Printf("Could not get provider for IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		if err := cl.Delete(obj.Status.ID); err != nil {
			log.Printf("Could not delete IngressMonitor %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}
	}
}

func (o *Operator) handleMonitor(obj *v1alpha1.Monitor) error {
	prov, err := o.imClient.Ingressmonitor().Providers(obj.Namespace).
		Get(obj.Spec.Provider.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	tmpl, err := o.imClient.Ingressmonitor().MonitorTemplates(obj.Namespace).
		Get(obj.Spec.Template.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	opts := metav1.ListOptions{
		LabelSelector: labels.FormatLabels(obj.Spec.Selector.MatchLabels),
	}
	ingressList, err := o.kubeClient.Extensions().Ingresses(obj.Namespace).List(opts)
	if err != nil {
		return err
	}

	// We'll calculate all the IngressMonitors that shouldn't be tracked
	// anymore and delete them. We can do this by fetching all
	// IngressMonitors where the owner is this Monitor, go over them all and
	// see if there are any where the Ingress Owner isn't in the new Ingress
	// List.
	imList, err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).
		List(metav1.ListOptions{
			LabelSelector: labels.FormatLabels(map[string]string{
				monitorLabel: obj.Name,
			}),
		})
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
			o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).
				Delete(im.Name, &metav1.DeleteOptions{})
		}
	}

	// reconcile the newly selected Ingresses. We'll create new IngressMonitors
	// for each Ingress and it's subsequent rules. If it already exists, we
	// update it.
	for _, ing := range ingressList.Items {
		for _, rule := range ing.Spec.Rules {
			im := &v1alpha1.IngressMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ing.Name,
					Namespace: ing.Namespace,
					// Add OwnerReferences to the IngressMonitor so we can
					// automatically Garbage Collect when either a Monitor is
					// removed or when the Ingress is removed. This way we don't
					// have to set this up ourselves.
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(
							obj,
							v1alpha1.SchemeGroupVersion.WithKind("Monitor"),
						),
						*metav1.NewControllerRef(
							&ing,
							extensions.SchemeGroupVersion.WithKind("Ingress"),
						),
					},
					// Set some labels so it's easier to filter later on
					Labels: map[string]string{
						monitorLabel:     obj.Name,
						ingressLabel:     ing.Name,
						ingressHostLabel: rule.Host,
					},
				},
				Spec: v1alpha1.IngressMonitorSpec{
					Provider: prov.Spec,
					Template: tmpl.Spec,
				},
			}

			if _, err := o.imClient.Ingressmonitor().IngressMonitors(obj.Namespace).Create(im); err != nil {
				return err
			}
		}
	}

	return nil
}
