package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderSpec is the detailed configuration for a Provider.
type ProviderSpec struct {
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
	Value *string `json:"value,omitempty"`

	// Optional: Specifies a source the value of this var should come from.
	// +optional
	ValueFrom *v1.SecretKeySelector `json:"valueFrom,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Provider is the CRD specification for an Provider. This
// Provider allows you to configure providers which will be used to set
// up monitors.
type Provider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ProviderSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderList is a list of Providers.
type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Provider `json:"items"`
}
