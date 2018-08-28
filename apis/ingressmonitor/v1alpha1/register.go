package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeBuilder collects the scheme builder functions for the
	// Monitor Custom Resources.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme applies the SchemeBuilder functions to a specified scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

const (
	// GroupName is the group name for the Monitor CRD.
	GroupName = "ingressmonitor.sphc.io"

	// APIVersion is the version for the API
	APIVersion = "v1alpha1"
)

// SchemeGroupVersion is the GroupVersion for the Monitor CRD.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: APIVersion}

// Resource gets an Monitor GroupResource for a specified resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Monitor{},
		&MonitorList{},
		&Provider{},
		&ProviderList{},
		&IngressMonitor{},
		&IngressMonitorList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
