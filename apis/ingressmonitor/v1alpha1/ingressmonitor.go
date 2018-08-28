package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IngressMonitorSpec is the detailed configuration for an Monitor.
type IngressMonitorSpec struct {
	// Provider describes the provider we want to use to set up the monitor
	// with.
	Provider ProviderSpec `json:"provider"`

	// Template describes the monitor configuration.
	Template MonitorTemplate `json:"template"`
}

// IngressMonitorStatus describes the status of an IngressMonitor. This is data
// which is used to handle Operator restarts or upgrades.
type IngressMonitorStatus struct {
	// ID describes the ID of the monitor which is registered with the provider.
	// This is used to update or delete the monitor with the provider.
	ID string `json:"id"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressMonitor is the detailed implementation of a Monitor which relates to
// a HTTP check. It's a fully qualified configuration which doesn't need to
// fetch any other data and can live on it's own.
// This can also be used to set up external monitors.
type IngressMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   IngressMonitorSpec   `json:"spec"`
	Status IngressMonitorStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressMonitorList is a list of Monitors
type IngressMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IngressMonitor `json:"items"`
}
