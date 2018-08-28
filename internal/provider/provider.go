package provider

import "github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"

// Interface reflects interface we'll use to speak with Monitoring Providers.
type Interface interface {
	Create(v1alpha1.MonitorTemplate) (string, error)
	Delete(string) error
	Update(string, v1alpha1.MonitorTemplate) error
}
