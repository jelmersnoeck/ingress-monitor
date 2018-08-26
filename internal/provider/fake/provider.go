package fake

import "github.com/jelmersnoeck/ingress-monitor/internal/provider"

// SimpleProvider represents a provider which is useful for testing purposes.
type SimpleProvider struct {
	CreateFunc func(provider.IngressMonitor) error
	DeleteFunc func(provider.IngressMonitor) error
	UpdateFunc func(provider.IngressMonitor) error
	ExistsFunc func(provider.IngressMonitor) (bool, error)
}

// CreateMonitor calls the specified CreateFunc in the SimpleProvider.
func (fp *SimpleProvider) CreateMonitor(im provider.IngressMonitor) error {
	return fp.CreateFunc(im)
}

// DeleteMonitor calls the specified DeleteFunc in the SimpleProvider.
func (fp *SimpleProvider) DeleteMonitor(im provider.IngressMonitor) error {
	return fp.DeleteFunc(im)
}

// UpdateMonitor calls the specified UpdateFunc in the SimpleProvider.
func (fp *SimpleProvider) UpdateMonitor(im provider.IngressMonitor) error {
	return fp.UpdateFunc(im)
}

// ExistsMonitor calls the specified ExistsFunc in the SimpleProvider.
func (fp *SimpleProvider) ExistsMonitor(im provider.IngressMonitor) (bool, error) {
	return fp.ExistsFunc(im)
}
