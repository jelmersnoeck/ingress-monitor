package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MonitorSpec is the detailed configuration for an Monitor.
type MonitorSpec struct {
	// Selector describes the LabelSelector which will be used to select the
	// enabled Ingresses which we want to set up monitors for.
	Selector metav1.LabelSelector `json:"selector"`

	// Provider describes the provider we want to use to set up the monitor
	// with.
	Provider v1.LocalObjectReference `json:"provider"`

	// Template describes the monitor configuration.
	Template MonitorTemplate `json:"template"`
}

// MonitorTemplate allow
type MonitorTemplate struct {
	// Type describes the type of check we want to use.
	Type string `json:"type"`

	// Name is the template that will be used to set the name of the check. It
	// follows the Go Template syntax.
	Name string `json:"name"`

	// CheckRate describes the number of seconds between checks. This defaults
	// to the provider's default.
	// +optional
	CheckRate *string `json:"checkRate,omitempty"`

	// Confirmations describes the amount of fails should occur before a check
	// is marked as a failure. This defaults to the provider's default.
	// +optional
	Confirmations *int `json:"confirmations,omitempty"`

	// Timeout describes the duration of how long a check should wait before
	// marking itself as unhealthy. Defaults to the provider's default.
	// +optional
	Timeout *string `json:"timeout,omitempty"`

	// HTTP is the template for a HTTP Check. This is required when the type is
	// set to `HTTP`.
	HTTP *HTTPTemplate `json:"http,omitempty"`
}

// HTTPTemplate describes the configuration options for a HTTP Check.
type HTTPTemplate struct {
	// URL describes the URL we want to check for the given website. Defaults to
	// `/_healthz`.
	// +optional
	URL *string `json:"url,omitempty"`

	// CustomHeader is a special header that will be sent along with the check
	// request. Defaults to the provider's default.
	// +optional
	CustomHeader string `json:"customHeader,omitempty"`

	// UserAgent describes the UserAgent that will be used to perform the check.
	// Defaults to the provider's default.
	// +optional
	UserAgent string `json:"userAgent,omitempty"`

	// VerifyCertificate specifies if the check should validate the SSL
	// Certificate. Defaults to false.
	// +optional
	VerifyCertificate bool `json:"verifyCertificate,omitempty"`

	// ShouldContain describes the string the response body should contain when
	// performing the check. Defaults to ``.
	// +optional
	ShouldContain string `json:"shouldContain,omitempty"`

	// ShouldNotContain describes the string which should not be present in the
	// response body when performing the check. Defaults to ``.
	// +optional
	ShouldNotContain string `json:"shouldNotContain,omitempty"`
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
