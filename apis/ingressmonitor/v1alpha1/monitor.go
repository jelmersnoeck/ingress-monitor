package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MonitorSpec is the detailed configuration for an Monitor.
type MonitorSpec struct {
	// Selector describes the LabelSelector which will be used to select the
	// enabled Ingresses which we want to set up monitors for.
	Selector *metav1.LabelSelector `json:"selector"`

	// Provider describes the provider we want to use to set up the monitor
	// with.
	Provider v1.LocalObjectReference `json:"provider"`

	// Template describes the monitor configuration.
	Template v1.LocalObjectReference `json:"template"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Monitor is the CRD specification for an Monitor. This
// Monitor allows you to configure monitors for the resources selected
// by it's configuration and instantiate them in the specified MonitorProvider.
type Monitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec MonitorSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitorList is a list of Monitors
type MonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Monitor `json:"items"`
}
