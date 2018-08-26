package provider

// Interface reflects interface we'll use to speak with Monitoring Providers.
type Interface interface {
	CreateMonitor(IngressMonitor) error
	DeleteMonitor(IngressMonitor) error
	UpdateMonitor(IngressMonitor) error
	ExistsMonitor(IngressMonitor) (bool, error)
}

// IngressMonitor describes the configuration for a Monitor which we'll use to
// send to a provider.
type IngressMonitor struct {
	// Type describes the type of check we want to use.
	Type string

	// Name is the fully qualified name for the IngressMonitor. This is the name
	// that will be used to create the monitor with the provider.
	Name string

	// CheckRate describes the number of seconds between checks. This defaults
	// to the provider's default.
	CheckRate *string

	// Confirmations describes the amount of fails should occur before a check
	// is marked as a failure. This defaults to the provider's default.
	Confirmations *int

	// Timeout describes the duration of how long a check should wait before
	// marking itself as unhealthy. Defaults to the provider's default.
	Timeout *string

	// Tags represents a set of tags for the given monitor. This is added to
	// make searching and filtering easier.
	Tags []string

	// HTTP is the template for a HTTP Check. This is required when the type is
	// set to `HTTP`.
	HTTP *HTTPTemplate
}

// HTTPTemplate describes the configuration options for a HTTP Check.
type HTTPTemplate struct {
	// URL describes the URL we want to check for the given website. Defaults to
	// `/_healthz`.
	URL *string

	// CustomHeader is a special header that will be sent along with the check
	// request. Defaults to the provider's default.
	CustomHeader string

	// UserAgent describes the UserAgent that will be used to perform the check.
	// Defaults to the provider's default.
	UserAgent string

	// VerifyCertificate specifies if the check should validate the SSL
	// Certificate. Defaults to false.
	VerifyCertificate bool

	// ShouldContain describes the string the response body should contain when
	// performing the check. Defaults to ``.
	ShouldContain string

	// ShouldNotContain describes the string which should not be present in the
	// response body when performing the check. Defaults to ``.
	ShouldNotContain string
}
