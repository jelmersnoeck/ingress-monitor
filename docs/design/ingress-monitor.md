# IngressMonitor

An IngressMonitor is the representation of a fully qualified Monitor. It has the
full configuration of the Provider it wishes to use and has a Template that is
completely filled out with all the details needed for the provider.

IngressMonitors can be installed on their own, but they are usually managed by
the Operator. Much like a ReplicaSet or Pod is managed by a Deployment.

When the Operator controls an IngressMonitor, it links it to a Monitor and
Ingress to ensure that when one of these objects gets removed from the cluster,
the IngressMonitor gets Garbage Collected as well.

```yaml
# The IngressMonitor object is what's used to configure a set of monitors for a
# selected set of resources.
apiVersion: ingressmonitor.sphc.io/v1alpha1
kind: IngressMonitor
metadata:
  name: go-apps
  namespace: websites
spec:
  # The provider is a fully qualified provider spec. This is the detailed
  # configuration which will be used to set up a client and configure the
  # template.
  provider:
    # Required. The type of provider used to
    type: StatusCake
    # The statusCake provider implementation. This will be required if type is
    # set to `StatusCake`.
    statusCake:
      # Required. The username to connect to StatusCake.
      username:
        # This could also be a valueFrom item. It's a subsection of the EnvVar
        # definition (https://godoc.org/k8s.io/kubernetes/pkg/apis/core#EnvVar),
        # minus the name (the name is the key here).
        value: jelmersnoeck
      # Required. The APIKey used to connect to StatusCake.
      apiKey:
        valueFrom:
          secretKeyRef:
            name: statuscake-secrets
            key: password
      # Optional. The ContactGroup ID that should be configured for the checks.
      contactGroups:
        - 1234567890
  # Requried. The actual configuration for the monitor.
  template:
    # Required. The type of check we want to perform.
    type: HTTP
    # Optional. The interval at which a monitor is triggered. Defaults to the
    # default of the provider.
    checkRate: 60s
    # Optional. The amount of confirmations needed by the configured provider
    # before it sends out an alert. This defaults to the default of the
    # configured provider.
    confirmations: 3
    # Required. Name template that will be used to configure the test. This
    # supports Go templates. Available values:
    # Name: the name of the selected Ingress
    # Namespace: the namespace of the selected Ingress
    name: {{.Name}}-{{.Namespace}}
    # Optional. The time after which the check will fail if there is no
    # response.
    timeout: 30s
    # Optional. This is required when the type is set to HTTP .
    http:
      # Optional. The URL which the configured provider should use to do it's
      # checks. Defaults to `/_healthz`.
      URL: /_healthz
      # Optional. A special header that will be sent along with your HTTP
      # Request. Defaults to ``.
      customHeader: "Custom-Header: IngressMonitor"
      # Optional. User agent used to populate the test. Defaults to ``.
      userAgent: "Siphoc IngressMonitor"
      # Optional. Allow the monitor to verify the SSL certificate. Defaults to
      # `false`.
      verifyCertificate: true
      # Optional. The target site should contain this string in the response
      # body. Defaults to ``.
      shouldContain: "OK"
      # Optional. The target site should not contain this string in the response
      # body. Defaults to ``.
      shouldNotContain: "Bad Gateway"
```
