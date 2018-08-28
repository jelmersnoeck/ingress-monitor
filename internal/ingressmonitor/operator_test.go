package ingressmonitor

import (
	"errors"
	"testing"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/fake"
	imfake "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		prov.CreateFunc = func(v1alpha1.MonitorTemplate) (string, error) {
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
		prov.CreateFunc = func(v1alpha1.MonitorTemplate) (string, error) {
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
		prov.UpdateFunc = func(status string, _ v1alpha1.MonitorTemplate) error {
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
		prov.UpdateFunc = func(status string, _ v1alpha1.MonitorTemplate) error {
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
