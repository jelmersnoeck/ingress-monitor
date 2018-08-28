# Provider

A provider allows you to configure a reusable component which will instantiate
a client when an [IngressMonitor](./ingress-monitor.md) is added to the cluster.

There are different types of providers, each containing their own configuration.

## StatusCake

A StatusCake Provider has 2 required fields, the `username` and `apiKey` which
is used to connect to StatusCake's API. As an optional argument, you can set up
a list of `contactGroups`. These contact groups are used within StatusCake to
send notifications to.

```yaml
apiVersion: ingressmonitor.sphc.io/v1alpha1
kind: MonitorProvider
metadata:
  name: prod-statuscake
  namespace: websites
spec:
  type: StatusCake
  statusCake:
    username:
      value: jelmersnoeck
    apiKey:
      valueFrom:
        secretKeyRef:
          name: statuscake-secrets
          key: password
    contactGroups:
      - 1234567890
```
