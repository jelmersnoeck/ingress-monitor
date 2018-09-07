package logger

import (
	"log"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"k8s.io/client-go/kubernetes"
)

// Register registers the provider with a certain factory using the FactoryFunc.
func Register(fact provider.FactoryInterface) {
	fact.Register("Logger", FactoryFunc)
}

// FactoryFunc is the function which will allow us to create clients on the fly
// which log out values.
func FactoryFunc(_ kubernetes.Interface, _ v1alpha1.NamespacedProvider) (provider.Interface, error) {
	return new(prov), nil
}

type prov struct{}

// Create logs out a create action.
func (p *prov) Create(ts v1alpha1.MonitorTemplateSpec) (string, error) {
	log.Printf("Creating monitor %s", ts.Name)

	return ts.Name, nil
}

// Delete logs out a delete action.
func (p *prov) Delete(id string) error {
	log.Printf("Deleting monitor %s", id)

	return nil
}

// Update logs out the update information for this template spec.
func (p *prov) Update(id string, ts v1alpha1.MonitorTemplateSpec) (string, error) {
	log.Printf("Updating monitor %s with ID %s", ts.Name, id)

	return id, nil
}
