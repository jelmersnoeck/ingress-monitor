package ingressmonitor

import (
	"errors"
	"testing"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/fake"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned"
	imfake "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/fake"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func TestOperator_HandleNextItem(t *testing.T) {
	op, _ := NewOperator(nil, nil, "", time.Minute, nil)
	var queue workqueue.RateLimitingInterface
	setup := func() {
		queue = workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(0, 0),
			"Tests",
		)
	}

	t.Run("with a shut down queue", func(t *testing.T) {
		setup()

		queue.ShutDown()

		if op.handleNextItem("test", queue, nil) {
			t.Errorf("Expected not to want to handle next item, got true")
		}
	})

	t.Run("with a non string object in the queue", func(t *testing.T) {
		setup()

		queue.Add(struct{}{})

		if queue.Len() != 1 {
			t.Fatalf("Expected 1 item to be added to the queue")
		}

		if !op.handleNextItem("test", queue, nil) {
			t.Errorf("Expected to want to proceed processing objects")
		}

		if queue.Len() != 0 {
			t.Errorf("Expected object to be removed from the queue")
		}
	})

	t.Run("with an error handling the object", func(t *testing.T) {
		setup()

		queue.Add("12345")

		if queue.Len() != 1 {
			t.Fatalf("Expected 1 item to be added to the queue")
		}

		handler := func(id string) error {
			if id != "12345" {
				t.Errorf("Expected ID to match, got %s", id)
			}

			return errors.New("Not handled!")
		}

		if !op.handleNextItem("test", queue, handler) {
			t.Errorf("Expected to want to proceed processing objects")
		}

		if queue.Len() != 0 {
			t.Errorf("Expected object to be removed from the queue")
		}
	})
}

func TestOperator_OnAddUpdate_IngressMonitor(t *testing.T) {
	t.Run("with error in handling func", func(t *testing.T) {
		// the handler errors when there is no provider, use that!
		crd := &v1alpha1.IngressMonitor{
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

		fact := provider.NewFactory(nil)
		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)
		op.ingressMonitorQueue = workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(0, 0),
			"IngressMonitors",
		)

		op.OnAdd(crd)

		if op.ingressMonitorQueue.Len() != 1 {
			t.Errorf("Expected 1 item in the queue, got %d", op.ingressMonitorQueue.Len())
		}

		// process the item
		if ok := op.processNextIngressMonitor(); ok {
			t.Errorf("Expected IngressMonitor not to be processed")
		}

		if op.ingressMonitorQueue.Len() != 0 {
			t.Errorf("Expected 0 items in the queue, got %d", op.ingressMonitorQueue.Len())
		}
	})

	t.Run("with everything set up", func(t *testing.T) {
		crd := &v1alpha1.IngressMonitor{
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

		prov := new(fake.SimpleProvider)
		prov.CreateFunc = func(v1alpha1.MonitorTemplateSpec) (string, error) {
			return "1234", nil
		}
		prov.UpdateFunc = func(id string, sp v1alpha1.MonitorTemplateSpec) (string, error) {
			return id, nil
		}

		fact := provider.NewFactory(nil)
		fact.Register("simple", fake.FactoryFunc(prov))

		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)
		op.ingressMonitorQueue = workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(0, 0),
			"IngressMonitors",
		)

		t.Run("add ingress monitor", func(t *testing.T) {
			op.OnAdd(crd)

			if op.ingressMonitorQueue.Len() != 1 {
				t.Errorf("Expected 1 item in the queue, got %d", op.ingressMonitorQueue.Len())
			}

			// process the item
			if ok := op.processNextIngressMonitor(); !ok {
				t.Errorf("Expected IngressMonitor to be processed")
			}

			if op.ingressMonitorQueue.Len() != 0 {
				t.Errorf("Expected 0 items in the queue, got %d", op.ingressMonitorQueue.Len())
			}
		})

		t.Run("update ingress monitor", func(t *testing.T) {
			op.OnUpdate(crd, crd)

			if op.ingressMonitorQueue.Len() != 1 {
				t.Errorf("Expected 1 item in the queue, got %d", op.ingressMonitorQueue.Len())
			}

			// process the item
			if ok := op.processNextIngressMonitor(); !ok {
				t.Errorf("Expected IngressMonitor to be processed")
			}

			if op.ingressMonitorQueue.Len() != 0 {
				t.Errorf("Expected 0 items in the queue, got %d", op.ingressMonitorQueue.Len())
			}
		})
	})
}

func Test_OnAddUpdate_Monitor(t *testing.T) {
	crd := &v1alpha1.Monitor{
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

	t.Run("with everything set up", func(t *testing.T) {
		prov := &v1alpha1.Provider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-provider",
				Namespace: "testing",
			},
		}
		tpl := &v1alpha1.MonitorTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-template",
				Namespace: "testing",
			},
			Spec: v1alpha1.MonitorTemplateSpec{
				Type: "HTTP",
				HTTP: &v1alpha1.HTTPTemplate{},
			},
		}

		k8sClient := k8sfake.NewSimpleClientset()
		crdClient := imfake.NewSimpleClientset(crd, prov, tpl)

		fact := provider.NewFactory(nil)
		op, _ := NewOperator(k8sClient, crdClient, "", time.Minute, fact)
		op.monitorQueue = workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(0, 0),
			"Monitors",
		)

		t.Run("add monitor", func(t *testing.T) {
			op.OnAdd(crd)

			if op.monitorQueue.Len() != 1 {
				t.Errorf("Expected 1 item in the queue, got %d", op.monitorQueue.Len())
			}

			// process the item
			if ok := op.processNextMonitor(); !ok {
				t.Errorf("Expected Monitor to be processed")
			}

			if op.monitorQueue.Len() != 0 {
				t.Errorf("Expected 0 items in the queue, got %d", op.monitorQueue.Len())
			}
		})

		t.Run("update monitor", func(t *testing.T) {
			op.OnUpdate(crd, crd)

			if op.monitorQueue.Len() != 1 {
				t.Errorf("Expected 1 item in the queue, got %d", op.ingressMonitorQueue.Len())
			}

			// process the item
			if ok := op.processNextMonitor(); !ok {
				t.Errorf("Expected Monitor to be processed")
			}

			if op.monitorQueue.Len() != 0 {
				t.Errorf("Expected 0 items in the queue, got %d", op.monitorQueue.Len())
			}
		})
	})
}

func TestOperator_HandleIngressMonitor(t *testing.T) {
	t.Run("creating a new test with the provider", func(t *testing.T) {
		fact := provider.NewFactory(nil)

		prov := new(fake.SimpleProvider)
		prov.CreateFunc = func(v1alpha1.MonitorTemplateSpec) (string, error) {
			return "1234", nil
		}

		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
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

		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)

		if err := op.handleIngressMonitor(namespaceKey(t, crd)); err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if prov.CreateCount != 1 {
			t.Errorf("Expected Create to be called once, got %d", prov.CreateCount)
		}

		crd, err := crdClient.Ingressmonitor().IngressMonitors(crd.Namespace).Get(crd.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Expected no error fetching the CRD, got %s", err)
		}

		if crd.Status.ID != "1234" {
			t.Errorf("Expected status to be updated")
		}
	})

	t.Run("with the item already deleted from the server", func(t *testing.T) {
		// this can happen when we had to queue for a while and the item has
		// been deleted in the meantime. We want to ensure it's handled
		// gracefully.
		fact := provider.NewFactory(nil)

		prov := new(fake.SimpleProvider)
		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
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

		// we'll never register the item, simulating that it's been deleted
		crdClient := imfake.NewSimpleClientset()
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)

		if err := op.handleIngressMonitor(namespaceKey(t, crd)); !kerrors.IsNotFound(err) {
			t.Errorf("Expected no error, got %s", err)
		}
	})

	t.Run("without registered provider", func(t *testing.T) {
		fact := provider.NewFactory(nil)

		prov := new(fake.SimpleProvider)
		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.NamespacedProvider{
					Namespace: "testing",
					ProviderSpec: v1alpha1.ProviderSpec{
						Type: "test",
					},
				},
			},
		}

		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)

		expErr := errors.New("Error fetching provider 'test': the specified provider can't be found")
		if err := op.handleIngressMonitor(namespaceKey(t, crd)); err.Error() != expErr.Error() {
			t.Errorf("Expected error '%s', got '%s'", expErr, err)
		}

		if prov.UpdateCount != 0 {
			t.Errorf("Expected no updates, got %d", prov.UpdateCount)
		}

		if prov.CreateCount != 0 {
			t.Errorf("Expected no updates, got %d", prov.UpdateCount)
		}
	})

	t.Run("with error updating monitor", func(t *testing.T) {
		fact := provider.NewFactory(nil)

		err := errors.New("my-provider-error")
		prov := new(fake.SimpleProvider)
		prov.UpdateFunc = func(status string, _ v1alpha1.MonitorTemplateSpec) (string, error) {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return status, err
		}

		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
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
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}

		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)

		if handleErr := op.handleIngressMonitor(namespaceKey(t, crd)); err != handleErr {
			t.Errorf("Expected error '%s', got '%s'", err, handleErr)
		}

		if prov.UpdateCount != 1 {
			t.Errorf("Expected Update to be called once, got %d", prov.UpdateCount)
		}
	})

	t.Run("without errors", func(t *testing.T) {
		fact := provider.NewFactory(nil)

		prov := new(fake.SimpleProvider)
		prov.UpdateFunc = func(status string, _ v1alpha1.MonitorTemplateSpec) (string, error) {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return status, nil
		}

		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
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
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}

		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, fact)

		if err := op.handleIngressMonitor(namespaceKey(t, crd)); err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if prov.UpdateCount != 1 {
			t.Errorf("Expected Update to be called once, got %d", prov.UpdateCount)
		}
	})
}

func TestOperator_OnDelete_IngressMonitor(t *testing.T) {
	t.Run("without registered provider", func(t *testing.T) {
		fact := provider.NewFactory(nil)
		op, _ := NewOperator(nil, nil, "", time.Minute, fact)

		prov := new(fake.SimpleProvider)
		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.NamespacedProvider{
					Namespace: "testing",
					ProviderSpec: v1alpha1.ProviderSpec{
						Type: "test",
					},
				},
			},
		}

		op.OnDelete(crd)

		if prov.DeleteCount != 0 {
			t.Errorf("Expected no deletes, got %d", prov.DeleteCount)
		}
	})

	t.Run("with error deleting monitor", func(t *testing.T) {
		fact := provider.NewFactory(nil)

		err := errors.New("my-provider-error")
		prov := new(fake.SimpleProvider)
		prov.DeleteFunc = func(status string) error {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return err
		}

		fact.Register("simple", fake.FactoryFunc(prov))
		op, _ := NewOperator(nil, nil, "", time.Minute, fact)

		crd := &v1alpha1.IngressMonitor{
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
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}

		op.OnDelete(crd)

		if prov.DeleteCount != 1 {
			t.Errorf("Expected delete to be called once, got %d", prov.DeleteCount)
		}
	})

	t.Run("without errors", func(t *testing.T) {
		fact := provider.NewFactory(nil)

		prov := new(fake.SimpleProvider)
		prov.DeleteFunc = func(status string) error {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return nil
		}

		fact.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
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
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}
		op, _ := NewOperator(nil, nil, "", time.Minute, fact)

		op.OnDelete(crd)

		if prov.DeleteCount != 1 {
			t.Errorf("Expected Update to be called once, got %d", prov.DeleteCount)
		}
	})
}

func TestOperator_HandleMonitor(t *testing.T) {
	ing1 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "go-ingress",
			Namespace: "testing",
			Labels: map[string]string{
				"team":  "gophers",
				"squad": "operations",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{Host: "api.example.com"},
			},
		},
	}
	ing2 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-ingress",
			Namespace: "testing",
			Labels: map[string]string{
				"team": "reacters",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{Host: "api.foo.com"},
			},
		},
	}

	prov := &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-provider",
			Namespace: "testing",
		},
	}
	tpl := &v1alpha1.MonitorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "testing",
		},
		Spec: v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{},
		},
	}

	mon := &v1alpha1.Monitor{
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

	var crdClient versioned.Interface
	var k8sClient kubernetes.Interface
	var op *Operator

	setup := func() {
		crdClient = imfake.NewSimpleClientset(prov, tpl)
		mon, _ = crdClient.Ingressmonitor().Monitors(mon.Namespace).Create(mon)

		k8sClient = k8sfake.NewSimpleClientset()
		ing1, _ = k8sClient.Extensions().Ingresses(ing1.Namespace).Create(ing1)
		ing2, _ = k8sClient.Extensions().Ingresses(ing2.Namespace).Create(ing2)

		op, _ = NewOperator(k8sClient, crdClient, v1.NamespaceAll, time.Minute, provider.NewFactory(nil))
		// we won't start the operator so the informers aren't automatically
		// trigerred. Make sure the monitor is added correctly.
		if err := op.handleMonitor(namespaceKey(t, mon)); err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}
	}

	t.Run("without CRD changes", func(t *testing.T) {
		t.Run("without Ingress changes", func(t *testing.T) {
			setup()
			if err := op.handleMonitor(namespaceKey(t, mon)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitor, got %d", len(imList.Items))
			}

			if !metav1.IsControlledBy(&imList.Items[0], ing1) {
				t.Errorf("Expected IngressMonitor to be owned by the correct Ingress")
			}
		})

		t.Run("with Ingress changes", func(t *testing.T) {
			setup()

			// the user has changed the Ingress and removed the labels in their
			// manifest
			ing := ing1.DeepCopy()
			ing.Labels = map[string]string{}
			k8sClient.Extensions().Ingresses(ing.Namespace).Update(ing)

			if err := op.handleMonitor(namespaceKey(t, mon)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			if len(imList.Items) != 0 {
				t.Errorf("Expected 0 IngressMonitor, got %d", len(imList.Items))
			}
		})

		t.Run("with Ingress additions", func(t *testing.T) {
			setup()

			// the user has changed the Ingress and removed the labels in their
			// manifest
			ing := ing2.DeepCopy()
			ing.Labels["team"] = "gophers"
			k8sClient.Extensions().Ingresses(ing.Namespace).Update(ing)

			if err := op.handleMonitor(namespaceKey(t, mon)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			if len(imList.Items) != 2 {
				t.Errorf("Expected 2 IngressMonitors, got %d", len(imList.Items))
			}
		})
	})

	t.Run("with CRD changes", func(t *testing.T) {
		t.Run("to change Ingresses", func(t *testing.T) {
			setup()

			new := mon.DeepCopy()
			new.Spec.Selector.MatchLabels["team"] = "reacters"
			if err := op.handleMonitor(namespaceKey(t, new)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitor, got %d", len(imList.Items))
			}

			if !metav1.IsControlledBy(&imList.Items[0], ing2) {
				t.Errorf("Expected IngressMonitor to be owned by the correct Ingress")
			}
		})

		t.Run("with the same Ingress", func(t *testing.T) {
			setup()

			new := mon.DeepCopy()
			// add a new label which makes selection more specific
			new.Spec.Selector.MatchLabels["squad"] = "operations"
			if err := op.handleMonitor(namespaceKey(t, new)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitor, got %d", len(imList.Items))
			}

			if !metav1.IsControlledBy(&imList.Items[0], ing1) {
				t.Errorf("Expected IngressMonitor to be owned by the correct Ingress")
			}
		})

		t.Run("with monitor already deleted", func(t *testing.T) {
			// this could be caused by another delete action, we want to ensure
			// that we handle this gracefully
			setup()

			err := crdClient.Ingressmonitor().Monitors(mon.Namespace).Delete(mon.Name, &metav1.DeleteOptions{})
			if err != nil {
				t.Fatalf("Could not delete Monitor: %s", err)
			}

			if err := op.handleMonitor(namespaceKey(t, mon)); !kerrors.IsNotFound(err) {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			// The IngressMonitor is still active, we haven't called the
			// OnDelete action yet
			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitors, got %d", len(imList.Items))
			}
		})

		t.Run("without matching Ingress", func(t *testing.T) {
			setup()

			// make sure we update the CRD in our fake store
			new := mon.DeepCopy()
			new.Spec.Selector.MatchLabels["non"] = "existing"
			new, err := crdClient.Ingressmonitor().Monitors(new.Namespace).Update(new)
			if err != nil {
				t.Fatalf("Could not update labels: %s", err)
			}

			if err := op.handleMonitor(namespaceKey(t, new)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Expected no error listing IngressMonitors, got %s", err)
			}

			if len(imList.Items) != 0 {
				t.Errorf("Expected 0 IngressMonitors, got %d", len(imList.Items))
			}
		})

		t.Run("with templating set up", func(t *testing.T) {
			prov := &v1alpha1.Provider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-provider",
					Namespace: "testing",
				},
			}

			tmpl := &v1alpha1.MonitorTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-template",
					Namespace: "testing",
				},
				Spec: v1alpha1.MonitorTemplateSpec{
					Name: "some-test-{{.IngressName}}-{{.IngressNamespace}}",
					Type: "HTTP",
					HTTP: &v1alpha1.HTTPTemplate{
						Endpoint: ptrString("/_healthz"),
					},
				},
			}

			mon := &v1alpha1.Monitor{
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

			crdClient := imfake.NewSimpleClientset(prov, tmpl, mon)
			op, _ := NewOperator(k8sClient, crdClient, v1.NamespaceAll, time.Minute, provider.NewFactory(nil))

			if err := op.handleMonitor(namespaceKey(t, mon)); err != nil {
				t.Fatalf("Expected no error, got %s", err)
			}

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).
				List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Could not get IngressMonitor List: %s", err)
			}

			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitor to be registered, got %d", len(imList.Items))
			}

			// check if the templated name is parsed
			expectedName := "some-test-go-ingress-testing"
			if name := imList.Items[0].Spec.Template.Name; name != expectedName {
				t.Errorf("Expected name to be `%s`, got `%s", expectedName, name)
			}
		})
	})

	t.Run("it should set up values correctly", func(t *testing.T) {
		ing := &v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "testing",
				Labels: map[string]string{
					"team": "gophers",
				},
			},
			Spec: v1beta1.IngressSpec{
				TLS: []v1beta1.IngressTLS{
					{
						Hosts: []string{
							"test-host.sphc.io",
						},
					},
				},
				Rules: []v1beta1.IngressRule{
					{Host: "test-host.sphc.io"},
				},
			},
		}

		k8sClient := k8sfake.NewSimpleClientset(ing)

		prov := &v1alpha1.Provider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-provider",
				Namespace: "testing",
			},
		}

		tmpl := &v1alpha1.MonitorTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-template",
				Namespace: "testing",
			},
			Spec: v1alpha1.MonitorTemplateSpec{
				Type: "HTTP",
				HTTP: &v1alpha1.HTTPTemplate{
					Endpoint: ptrString("/_healthz"),
				},
			},
		}

		mon := &v1alpha1.Monitor{
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

		crdClient := imfake.NewSimpleClientset(prov, tmpl, mon)
		op, _ := NewOperator(k8sClient, crdClient, v1.NamespaceAll, time.Minute, provider.NewFactory(nil))

		if err := op.handleMonitor(namespaceKey(t, mon)); err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).
			List(metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Could not get IngressMonitor List: %s", err)
		}

		if len(imList.Items) != 1 {
			t.Errorf("Expected 1 IngressMonitor, got %d", len(imList.Items))
		}

		im := imList.Items[0]
		expURL := "https://test-host.sphc.io/_healthz"
		if url := im.Spec.Template.HTTP.URL; url != expURL {
			t.Errorf("Expected URL to be `%s`, got `%s`", expURL, url)
		}
	})
}

func ptrString(s string) *string {
	return &s
}

func namespaceKey(t *testing.T, obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		t.Fatalf("Could not get NamespaceKey for object %#v", obj)
	}

	return key
}
