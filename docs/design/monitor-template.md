# MonitorTemplate

A Monitor is used to set up a high level configuration for a set of checks. It
can be used to configure multiple Ingresses by setting up a selector.

For each Ingress that matches the selector, the Operator creates an
[IngressMonitor](./ingress-monitor.md).

The aim of a MonitorTemplate is reusability. Combine a MonitorTemplate with a
Provider and you have a [Monitor](./monitor.md)

```yaml
# A MonitorTemplate is a reusable configuration for a type of monitor. It can be
# referenced by a Monitor and can be reused for different types of Providers.
apiVersion: ingressmonitor.sphc.io/v1alpha1
kind: MonitorTemplate
metadata:
  name: go-apps
  namespace: websites
spec:
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
    # Optional. The endpoint which the configured provider should use to do it's
    # checks. Defaults to `/_healthz`.
    endpoint: `/_healthz`
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
