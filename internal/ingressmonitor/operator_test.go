package ingressmonitor

import (
	"errors"
	"testing"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/fake"
	imfake "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/fake"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestOperator_OnAdd_IngressMonitor(t *testing.T) {
	t.Run("without registered provider", func(t *testing.T) {
		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "test",
				},
			},
		}

		op.OnAdd(crd)
	})

	t.Run("with error creating monitor", func(t *testing.T) {
		defer provider.DefaultFactory().Flush()

		err := errors.New("my-provider-error")
		prov := new(fake.SimpleProvider)
		prov.CreateFunc = func(v1alpha1.MonitorTemplateSpec) (string, error) {
			return "", err
		}

		provider.Register("simple", fake.FactoryFunc(prov))

		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "simple",
				},
			},
		}

		op.OnAdd(crd)

		if prov.CreateCount != 1 {
			t.Errorf("Expected Create to be called once, got %d", prov.CreateCount)
		}
	})

	t.Run("without errors", func(t *testing.T) {
		defer provider.DefaultFactory().Flush()

		prov := new(fake.SimpleProvider)
		prov.CreateFunc = func(v1alpha1.MonitorTemplateSpec) (string, error) {
			return "1234", nil
		}

		provider.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "simple",
				},
			},
		}
		crdClient := imfake.NewSimpleClientset(crd)
		op, _ := NewOperator(nil, crdClient, "", time.Minute, provider.DefaultFactory())

		op.OnAdd(crd)

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

	t.Run("with ID already set", func(t *testing.T) {
		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		prov := new(fake.SimpleProvider)
		provider.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "test",
				},
			},
			Status: v1alpha1.IngressMonitorStatus{
				ID: "1234",
			},
		}

		op.OnAdd(crd)

		if prov.CreateCount != 0 {
			t.Errorf("Did not expect an object to be created, got a create call")
		}
	})
}

func TestOperator_OnUpdate_IngressMonitor(t *testing.T) {
	t.Run("without registered provider", func(t *testing.T) {
		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		prov := new(fake.SimpleProvider)
		provider.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "test",
				},
			},
		}

		op.OnUpdate(crd, crd)

		if prov.UpdateCount != 0 {
			t.Errorf("Expected no updates, got %d", prov.UpdateCount)
		}
	})

	t.Run("with error updating monitor", func(t *testing.T) {
		defer provider.DefaultFactory().Flush()

		err := errors.New("my-provider-error")
		prov := new(fake.SimpleProvider)
		prov.UpdateFunc = func(status string, _ v1alpha1.MonitorTemplateSpec) error {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return err
		}

		provider.Register("simple", fake.FactoryFunc(prov))

		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "simple",
				},
			},
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}

		op.OnUpdate(crd, crd)

		if prov.UpdateCount != 1 {
			t.Errorf("Expected Update to be called once, got %d", prov.UpdateCount)
		}
	})

	t.Run("without errors", func(t *testing.T) {
		defer provider.DefaultFactory().Flush()

		prov := new(fake.SimpleProvider)
		prov.UpdateFunc = func(status string, _ v1alpha1.MonitorTemplateSpec) error {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return nil
		}

		provider.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "simple",
				},
			},
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}
		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		op.OnUpdate(crd, crd)

		if prov.UpdateCount != 1 {
			t.Errorf("Expected Update to be called once, got %d", prov.UpdateCount)
		}
	})
}

func TestOperator_OnDelete_IngressMonitor(t *testing.T) {
	t.Run("without registered provider", func(t *testing.T) {
		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		prov := new(fake.SimpleProvider)
		provider.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "test",
				},
			},
		}

		op.OnDelete(crd)

		if prov.DeleteCount != 0 {
			t.Errorf("Expected no deletes, got %d", prov.DeleteCount)
		}
	})

	t.Run("with error deleting monitor", func(t *testing.T) {
		defer provider.DefaultFactory().Flush()

		err := errors.New("my-provider-error")
		prov := new(fake.SimpleProvider)
		prov.DeleteFunc = func(status string) error {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return err
		}

		provider.Register("simple", fake.FactoryFunc(prov))

		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "simple",
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
		defer provider.DefaultFactory().Flush()

		prov := new(fake.SimpleProvider)
		prov.DeleteFunc = func(status string) error {
			if status != "12345" {
				t.Errorf("Expected status to be `12345`, got `%s`", status)
			}
			return nil
		}

		provider.Register("simple", fake.FactoryFunc(prov))

		crd := &v1alpha1.IngressMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-im",
				Namespace: "testing",
			},
			Spec: v1alpha1.IngressMonitorSpec{
				Provider: v1alpha1.ProviderSpec{
					Type: "simple",
				},
			},
			Status: v1alpha1.IngressMonitorStatus{
				ID: "12345",
			},
		}
		op, _ := NewOperator(nil, nil, "", time.Minute, provider.DefaultFactory())

		op.OnDelete(crd)

		if prov.DeleteCount != 1 {
			t.Errorf("Expected Update to be called once, got %d", prov.DeleteCount)
		}
	})
}

func Test_OnAdd_Monitor(t *testing.T) {
	t.Run("without matching ingresses", func(t *testing.T) {
		ing := &v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "testing",
				Labels: map[string]string{
					"service": "no-match",
				},
			},
		}

		k8sClient := k8sfake.NewSimpleClientset(ing)
		crdClient := imfake.NewSimpleClientset()

		op, _ := NewOperator(k8sClient, crdClient, v1.NamespaceAll, time.Minute, provider.DefaultFactory())

		mon := &v1alpha1.Monitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-monitor",
				Namespace: "testing",
			},
			Spec: v1alpha1.MonitorSpec{
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"team": "gophers",
					},
				},
			},
		}

		op.OnAdd(mon)

		imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).
			List(metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Could not get IngressMonitor List: %s", err)
		}

		if len(imList.Items) != 0 {
			t.Errorf("Expected no IngressMonitors to be registered, got %d", len(imList.Items))
		}
	})

	t.Run("with matching ingresses", func(t *testing.T) {
		ing := &v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "testing",
				Labels: map[string]string{
					"team": "gophers",
				},
			},
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						Host: "test-host.sphc.io",
					},
				},
			},
		}

		k8sClient := k8sfake.NewSimpleClientset(ing)

		t.Run("without provider configured", func(t *testing.T) {
			crdClient := imfake.NewSimpleClientset()
			op, _ := NewOperator(k8sClient, crdClient, v1.NamespaceAll, time.Minute, provider.DefaultFactory())

			mon := &v1alpha1.Monitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-monitor",
					Namespace: "testing",
				},
				Spec: v1alpha1.MonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"team": "gophers",
						},
					},
					Provider: v1.LocalObjectReference{
						Name: "test-provider",
					},
				},
			}

			op.OnAdd(mon)

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).
				List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Could not get IngressMonitor List: %s", err)
			}

			if len(imList.Items) != 0 {
				t.Errorf("Expected 0 IngressMonitor to be registered, got %d", len(imList.Items))
			}
		})

		t.Run("with everything configured", func(t *testing.T) {
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
			}

			crdClient := imfake.NewSimpleClientset(prov, tmpl)
			op, _ := NewOperator(k8sClient, crdClient, v1.NamespaceAll, time.Minute, provider.DefaultFactory())

			mon := &v1alpha1.Monitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-monitor",
					Namespace: "testing",
				},
				Spec: v1alpha1.MonitorSpec{
					Selector: metav1.LabelSelector{
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

			op.OnAdd(mon)

			imList, err := crdClient.Ingressmonitor().IngressMonitors(mon.Namespace).
				List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Could not get IngressMonitor List: %s", err)
			}

			if len(imList.Items) != 1 {
				t.Errorf("Expected 1 IngressMonitor to be registered, got %d", len(imList.Items))
			}
		})
	})
}
