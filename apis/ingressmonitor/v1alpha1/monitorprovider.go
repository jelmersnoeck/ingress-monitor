package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MonitorProviderSpec is the detailed configuration for a MonitorProvider.
type MonitorProviderSpec struct {
	// Type describes the type of Provider which this CRD will configure.
	Type string `json:"type"`

	// StatusCake describes the StatusCake Monitoring Provider
	// +optional
	StatusCake *StatusCakeProvider `json:"statusCake,omitempty"`
}

// StatusCakeProvider describes the configuration options for the StatusCake
// provider.
type StatusCakeProvider struct {
	// Username is the username used to connect to StatusCake.
	Username SecretVar `json:"username"`

	// APIKey is the API Key used to connect to StatusCake.
	APIKey SecretVar `json:"apiKey"`

	// Optional: ContactGroups is a list of IDs which describes the groups which
	// should be alerted when a monitor check fails.
	// +optional
	ContactGroups []string `json:"contactGroups,omitempty"`
}

// SecretVar describes a secret var option which can be used to either provide
// a plaintext value or a secret value.
type SecretVar struct {
	// Optional: Specifies a plaintext value of
	// +optional
	Value string

	// Optional: Specifies a source the value of this var should come from.
	// +optional
	ValueFrom *v1.EnvVarSource
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitorProvider is the CRD specification for an MonitorProvider. This
// MonitorProvider allows you to configure providers which will be used to set
// up monitors.
type MonitorProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec MonitorProviderSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitorProviderList is a list of MonitorProviders.
type MonitorProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []MonitorProvider `json:"items"`
}
