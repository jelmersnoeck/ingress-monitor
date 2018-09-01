# Provider

A provider allows you to configure a reusable component which will instantiate
a client when an [IngressMonitor](./ingress-monitor.md) is added to the cluster.

There are different types of providers, each containing their own configuration.

A provider is namespace scoped as it can reference Secrets and ConfigMaps. These
Secrets and ConfigMaps need to live in the same namespace as the Provider.

## StatusCake

A StatusCake Provider has 2 required fields, the `username` and `apiKey` which
is used to connect to StatusCake's API. As an optional argument, you can set up
a list of `contactGroups`. These contact groups are used within StatusCake to
send notifications to.

```yaml
# A MonitorProvider is used to set up configuration for a specific monitoring
# provider.
apiVersion: ingressmonitor.sphc.io/v1alpha1
kind: Provider
metadata:
  name: prod-statuscake
  namespace: websites
spec:
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
```
