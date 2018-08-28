package ingressmonitor

import (
	"log"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned"
	crdscheme "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/scheme"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/informers/externalversions"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
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
}

// OnUpdate handles updates of IngressMonitors anad Ingresses and configures the
// checks with the configured providers.
func (o *Operator) OnUpdate(old, new interface{}) {
}

// OnDelete handles deletion of IngressMonitors and Ingresses and deletes
// monitors from the configured providers.
func (o *Operator) OnDelete(obj interface{}) {}
