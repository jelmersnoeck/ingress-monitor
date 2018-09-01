package fake

import (
	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"k8s.io/client-go/kubernetes"
)

// SimpleProvider represents a provider which is useful for testing purposes.
type SimpleProvider struct {
	CreateFunc  func(v1alpha1.MonitorTemplateSpec) (string, error)
	CreateCount int

	DeleteFunc  func(string) error
	DeleteCount int

	UpdateFunc  func(string, v1alpha1.MonitorTemplateSpec) (string, error)
	UpdateCount int
}

// Create calls the specified CreateFunc in the SimpleProvider.
func (fp *SimpleProvider) Create(im v1alpha1.MonitorTemplateSpec) (string, error) {
	fp.CreateCount++
	return fp.CreateFunc(im)
}

// Delete calls the specified DeleteFunc in the SimpleProvider.
func (fp *SimpleProvider) Delete(id string) error {
	fp.DeleteCount++
	return fp.DeleteFunc(id)
}

// Update calls the specified UpdateFunc in the SimpleProvider.
func (fp *SimpleProvider) Update(id string, im v1alpha1.MonitorTemplateSpec) (string, error) {
	fp.UpdateCount++
	return fp.UpdateFunc(id, im)
}

// FactoryFunc is used to register the factory in a given test so we can use it
// to test provider calls.
func FactoryFunc(sp *SimpleProvider) provider.FactoryFunc {
	return func(kubernetes.Interface, v1alpha1.NamespacedProvider) (provider.Interface, error) {
		return sp, nil
	}
}
