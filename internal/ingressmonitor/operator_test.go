package ingressmonitor

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/fake"
	imfake "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/fake"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func TestOperator_RunShutdown(t *testing.T) {
	t.Run("with cache sync error", func(t *testing.T) {
		op := newOperator(t).op

		stopCh := make(chan struct{})

		go func() {
			// by closing the channel prematurely, we make sure the cache syncs
			// fail
			close(stopCh)
		}()

		errEquals(t, errCouldNotSyncCache, op.Run(stopCh), "shutting down the operator")
	})

	t.Run("after the caches have been synced", func(t *testing.T) {
		op := newOperator(t).op

		stopCh := make(chan struct{})

		go func() {
			for _, inf := range op.informers {
				cache.WaitForCacheSync(stopCh, inf.informer.HasSynced)
			}

			close(stopCh)
		}()

		errEquals(t, nil, op.Run(stopCh), "shutting down the operator")
	})
}

func TestOperator_DeleteIngressMonitor(t *testing.T) {
	t.Run("delete the monitor with the provider", func(t *testing.T) {
		im := newIngressMonitor()
		im.Status.ID = "12345"
		op := newOperator(t,
			withIngressMonitors(im),
			withIngresses(newIngress()),
			withProviders(newProvider()),
			withTemplates(newTemplate()),
		)

		prov := new(fake.SimpleProvider)
		op.op.providerFactory.Register("simple", fake.FactoryFunc(prov))

		prov.DeleteFunc = func(id string) error {
			strEquals(t, "12345", id, "deleting the IngressMonitor")
			return nil
		}

		op.op.OnDelete(im)

		if prov.DeleteCount != 1 {
			t.Errorf("Expected the delete action to be called")
		}
	})
}

func TestOperator_DeleteMonitor(t *testing.T) {
	t.Run("delete all associated IngressMonitors", func(t *testing.T) {
		op := newOperator(t,
			withIngresses(newIngress()),
			withProviders(newProvider()),
			withTemplates(newTemplate()),
		)

		mon := newMonitor()
		errEquals(t, nil, op.handleMonitor(t, mon), "creating a new monitor")

		imList, err := op.op.imClient.IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
		errEquals(t, nil, err, "listing the IngressMonitors")

		if len(imList.Items) != 1 {
			t.Errorf("Expected 1 IngressMonitor to be available")
		}

		op.op.OnDelete(mon)

		imList, err = op.op.imClient.IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
		errEquals(t, nil, err, "listing the IngressMonitors")

		if len(imList.Items) != 0 {
			t.Errorf("Expected 0 IngressMonitor to be available")
		}
	})
}

func TestOperator_SyncIngressMonitor(t *testing.T) {
	t.Run("without configured provider", func(t *testing.T) {
		op := newOperator(t)

		im := newIngressMonitor()
		expError := fmt.Errorf("Error fetching provider 'simple': the specified provider can't be found")
		errEquals(t, expError, op.handleIngressMonitor(t, im))
	})

	t.Run("with enqueued item already deleted", func(t *testing.T) {
		op := newOperator(t)

		im := newIngressMonitor()
		// call the operator handleIngressMonitor function directly, bypassing
		// adding data to the cache
		errEquals(t, nil, op.op.handleIngressMonitor(getKey(t, im)))
	})

	t.Run("with provider configured", func(t *testing.T) {
		var op *operatorWrapper
		var prov *fake.SimpleProvider

		setup := func() {
			op = newOperator(t)
			prov = new(fake.SimpleProvider)
			op.op.providerFactory.Register("simple", fake.FactoryFunc(prov))
		}

		t.Run("with a new ingress monitor", func(t *testing.T) {
			t.Run("without an error", func(t *testing.T) {
				setup()

				prov.CreateFunc = func(tpl v1alpha1.MonitorTemplateSpec) (string, error) {
					return "12345", nil
				}

				im := newIngressMonitor()
				errEquals(t, nil, op.handleIngressMonitor(t, im), "adding an ingress monitor")

				im, err := op.op.imClient.IngressMonitors(im.Namespace).Get(im.Name, metav1.GetOptions{})
				errEquals(t, nil, err, "getting updated IngressMonitor")

				strEquals(t, "12345", im.Status.ID, "status should be the same")
			})

			t.Run("without an error", func(t *testing.T) {
				setup()

				expErr := errors.New("can't create monitor")
				prov.CreateFunc = func(tpl v1alpha1.MonitorTemplateSpec) (string, error) {
					return "12345", expErr
				}

				im := newIngressMonitor()
				errEquals(t, expErr, op.handleIngressMonitor(t, im), "adding an ingress monitor")
			})
		})

		t.Run("resyncing an existing ingress monitor", func(t *testing.T) {
			t.Run("without an error", func(t *testing.T) {
				setup()

				prov.UpdateFunc = func(id string, tpl v1alpha1.MonitorTemplateSpec) (string, error) {
					strEquals(t, "12345", id, "id to update")

					return "123456", nil
				}

				im := newIngressMonitor()
				im.Status.ID = "12345"
				errEquals(t, nil, op.handleIngressMonitor(t, im), "updating an ingress monitor")

				im, err := op.op.imClient.IngressMonitors(im.Namespace).Get(im.Name, metav1.GetOptions{})
				errEquals(t, nil, err, "getting updated IngressMonitor")

				strEquals(t, "123456", im.Status.ID, "status should be the same")
			})

			t.Run("without an error", func(t *testing.T) {
				setup()

				expErr := errors.New("can't create monitor")
				prov.UpdateFunc = func(id string, tpl v1alpha1.MonitorTemplateSpec) (string, error) {
					strEquals(t, "12345", id, "id to update")

					return id, expErr
				}

				im := newIngressMonitor()
				im.Status.ID = "12345"
				errEquals(t, expErr, op.handleIngressMonitor(t, im), "updating an ingress monitor")
			})
		})
	})
}

func TestOperator_SyncMonitor(t *testing.T) {
	t.Run("without matching ingresses", func(t *testing.T) {
		op := newOperator(t)

		mon := newMonitor()
		errEquals(t, nil, op.handleMonitor(t, mon))
	})

	t.Run("without existing provider", func(t *testing.T) {
		op := newOperator(t, withIngresses(newIngress()))

		mon := newMonitor()
		expError := fmt.Errorf("Could not get Provider testing:test-provider: provider.ingressmonitor.sphc.io \"test-provider\" not found")
		errEquals(t, expError, op.handleMonitor(t, mon))
	})

	t.Run("without existing template", func(t *testing.T) {
		op := newOperator(t,
			withIngresses(newIngress()),
			withProviders(newProvider()),
		)

		mon := newMonitor()
		expError := fmt.Errorf("Could not get MonitorTemplate test-template: monitortemplate.ingressmonitor.sphc.io \"test-template\" not found")
		errEquals(t, expError, op.handleMonitor(t, mon))
	})

	t.Run("with an ingress provider and template should create an IngressMonitor", func(t *testing.T) {
		op := newOperator(t,
			withIngresses(newIngress()),
			withProviders(newProvider()),
			withTemplates(newTemplate()),
		)

		stopCh := make(chan struct{})
		op.op.startInformers(stopCh)

		defer func() {
			stopCh <- struct{}{}
		}()

		mon := newMonitor()
		errEquals(t, nil, op.handleMonitor(t, mon), "creating a new monitor")

		imList, err := op.op.imClient.IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
		errEquals(t, nil, err, "listing the IngressMonitors")

		if len(imList.Items) != 1 {
			t.Errorf("Expected 1 IngressMonitor to be created")
		}

		im := imList.Items[0]

		expName := "test-go-ingress-testing"
		strEquals(t, expName, im.Spec.Template.Name)

		expURL := "https://api.example.com/test-healthz"
		strEquals(t, expURL, im.Spec.Template.HTTP.URL)
	})

	t.Run("updating an existing monitor", func(t *testing.T) {
		var op *operatorWrapper
		var stopCh chan struct{}
		setup := func() {
			op = newOperator(t,
				withIngresses(newIngress()),
				withProviders(newProvider()),
				withTemplates(newTemplate()),
			)

			// ensure that the monitor is added corectly
			errEquals(t, nil, op.handleMonitor(t, newMonitor()))

			stopCh = make(chan struct{})
			op.op.startInformers(stopCh)
		}

		cleanup := func() {
			stopCh <- struct{}{}
		}

		t.Run("changing labels", func(t *testing.T) {
			setup()
			defer cleanup()

			mon := newMonitor()

			imList, err := op.op.imClient.IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			errEquals(t, nil, err)

			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitor to be available, got %d", len(imList.Items))
			}

			mon.Spec.Selector.MatchLabels["non-existing-key"] = "fake-value"
			errEquals(t, nil, op.handleMonitor(t, mon))

			imList, err = op.op.imClient.IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			errEquals(t, nil, err)

			if len(imList.Items) != 0 {
				t.Errorf("Expected 0 IngressMonitor to be available, got %d", len(imList.Items))
			}
		})

		t.Run("adding an ingress and resyncing", func(t *testing.T) {
			setup()
			defer cleanup()

			ing := newIngress()
			ing.Name = "new-ingress"
			newRule := v1beta1.IngressRule{
				Host: "new.api.example.com",
			}
			ing.Spec.Rules = append(ing.Spec.Rules, newRule)
			op.addIngress(ing)

			// trigger resync
			mon := newMonitor()
			errEquals(t, nil, op.handleMonitor(t, mon))

			imList, err := op.op.imClient.IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			errEquals(t, nil, err)

			if len(imList.Items) != 3 {
				t.Errorf("Expected 3 IngressMonitors to be available, got %d", len(imList.Items))
			}
		})
	})
}

type operatorWrapper struct {
	op         *Operator
	kubeClient *k8sfake.Clientset
	imClient   *imfake.Clientset
}

type operatorConfig struct {
	ingresses   []runtime.Object
	kubeObjects []runtime.Object

	providers       []runtime.Object
	templates       []runtime.Object
	monitors        []runtime.Object
	ingressmonitors []runtime.Object
	crdObjects      []runtime.Object
}

type optionFunc func(*operatorConfig)

func withIngresses(obj ...runtime.Object) optionFunc {
	return func(op *operatorConfig) {
		op.ingresses = append(op.ingresses, obj...)
		op.kubeObjects = append(op.kubeObjects, obj...)
	}
}

func withProviders(obj ...runtime.Object) optionFunc {
	return func(op *operatorConfig) {
		op.providers = append(op.providers, obj...)
		op.crdObjects = append(op.crdObjects, obj...)
	}
}

func withTemplates(obj ...runtime.Object) optionFunc {
	return func(op *operatorConfig) {
		op.templates = append(op.templates, obj...)
		op.crdObjects = append(op.crdObjects, obj...)
	}
}

func withMonitors(obj ...runtime.Object) optionFunc {
	return func(op *operatorConfig) {
		op.monitors = append(op.monitors, obj...)
		op.crdObjects = append(op.crdObjects, obj...)
	}
}

func withIngressMonitors(obj ...runtime.Object) optionFunc {
	return func(op *operatorConfig) {
		op.ingressmonitors = append(op.ingressmonitors, obj...)
		op.crdObjects = append(op.crdObjects, obj...)
	}
}

func newOperator(t *testing.T, opts ...optionFunc) *operatorWrapper {
	cfg := new(operatorConfig)
	for _, opt := range opts {
		opt(cfg)
	}

	k8sClient := k8sfake.NewSimpleClientset(cfg.kubeObjects...)
	crdClient := imfake.NewSimpleClientset(cfg.crdObjects...)
	fact := provider.NewFactory(nil)
	op, err := NewOperator(k8sClient, crdClient, v1.NamespaceAll, noResyncPeriodFunc(), fact)
	if err != nil {
		t.Fatalf("Error creating the operator: %s", err)
	}

	op.ingressMonitorQueue = workqueue.NewNamedRateLimitingQueue(
		workqueue.NewItemExponentialFailureRateLimiter(0, 0),
		"IngressMonitors",
	)
	op.monitorQueue = workqueue.NewNamedRateLimitingQueue(
		workqueue.NewItemExponentialFailureRateLimiter(0, 0),
		"Monitors",
	)

	for _, ing := range cfg.ingresses {
		op.ingInformer.GetIndexer().Add(ing)
	}

	for _, prov := range cfg.providers {
		op.provInformer.GetIndexer().Add(prov)
	}

	for _, tpl := range cfg.templates {
		op.mtInformer.GetIndexer().Add(tpl)
	}

	for _, im := range cfg.ingressmonitors {
		op.imInformer.GetIndexer().Add(im)
	}

	return &operatorWrapper{op, k8sClient, crdClient}
}

func (o *operatorWrapper) handleIngressMonitor(t *testing.T, mon *v1alpha1.IngressMonitor) error {
	o.op.imInformer.GetIndexer().Add(mon)
	o.op.imClient.IngressMonitors(mon.Namespace).Create(mon)
	return o.op.handleIngressMonitor(getKey(t, mon))
}

func (o *operatorWrapper) handleMonitor(t *testing.T, mon *v1alpha1.Monitor) error {
	o.op.mInformer.GetIndexer().Add(mon)
	o.op.imClient.Monitors(mon.Namespace).Create(mon)
	return o.op.handleMonitor(getKey(t, mon))
}

func (o *operatorWrapper) addIngress(ing *v1beta1.Ingress) {
	o.op.ingInformer.GetIndexer().Add(ing)
}

var noResyncPeriodFunc = func() time.Duration { return 0 }

func getKey(t *testing.T, obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		t.Fatalf("Could not get namespaced key for %#v:\n\n%s", obj, err)
	}

	return key
}

func strEquals(t *testing.T, exp, act string, str ...string) {
	prefix := ""
	for _, s := range str {
		prefix = fmt.Sprintf("%s%s ", prefix, s)
	}

	if exp != act {
		t.Errorf("%sxpected value to be '%s', got '%s'", prefix, exp, act)
	}
}

func errEquals(t *testing.T, exp, act error, str ...string) {
	prefix := ""
	for _, s := range str {
		prefix = fmt.Sprintf("%s%s ", prefix, s)
	}

	if exp == nil && act == nil {
		return
	}

	if exp != nil && act == nil {
		t.Fatalf("%sexpected error %s, got none", prefix, exp)
	}

	if exp == nil && act != nil {
		t.Fatalf("%sexpected no error, got %s", prefix, act)
	}

	if exp.Error() != act.Error() {
		t.Fatalf("%sexpected error \n%s\ngot\n%s\n", prefix, exp, act)
	}
}

func newTemplate() *v1alpha1.MonitorTemplate {
	return &v1alpha1.MonitorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "testing",
		},
		Spec: v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{
				Endpoint: ptrString("/test-healthz"),
			},
			Name: "test-{{.IngressName}}-{{.IngressNamespace}}",
		},
	}
}

func newMonitor() *v1alpha1.Monitor {
	return &v1alpha1.Monitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-monitor",
			Namespace: "testing",
		},
		Spec: v1alpha1.MonitorSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"team": "gophers",
				},
			},
			Provider: v1.LocalObjectReference{
				Name: "test-provider",
			},
			Template: v1.LocalObjectReference{
				Name: "test-template",
			},
		},
	}
}

func newIngress() *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "go-ingress",
			Namespace: "testing",
			Labels: map[string]string{
				"team":  "gophers",
				"squad": "operations",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						"api.example.com",
					},
				},
			},
			Rules: []v1beta1.IngressRule{
				{Host: "api.example.com"},
			},
		},
	}
}

func newProvider() *v1alpha1.Provider {
	return &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-provider",
			Namespace: "testing",
		},
	}
}

func newIngressMonitor() *v1alpha1.IngressMonitor {
	return &v1alpha1.IngressMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-im",
			Namespace: "testing",
		},
		Spec: v1alpha1.IngressMonitorSpec{
			Provider: v1alpha1.NamespacedProvider{
				Namespace: "testing",
				ProviderSpec: v1alpha1.ProviderSpec{
					Type: "simple",
				},
			},
		},
	}
}

func ptrString(s string) *string {
	return &s
}
