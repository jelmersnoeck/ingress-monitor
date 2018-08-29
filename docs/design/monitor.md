# Monitor

A monitor is fairly simple as it's aim is to combine configuration objects into
a single object.

A Monitor is namespace bound, this means that the namespace you deploy it in is
where it will have impact. It won't select Ingresses from outside of that
namespace.

```yaml
# A Monitor is the glue between a MonitorTemplate, Provider and a set of
# Ingresses.
apiVersion: ingressmonitor.sphc.io/v1alpha1
kind: Monitor
metadata:
  name: go-apps
  namespace: websites
spec:
  # Required. The Operator will fetch all Ingresses that have the given labels
  # set up for the namespace this IngressMonitor lives in.
  selector:
    labels:
      component: marketplace
  # Provider is the provider we'd like to use for this Monitor.
  provider:
    name: prod-statuscake
  # Template is the reference to the MonitorTemplate we'd like to use for this
  # Monitor.
  template:
    name: go-apps
```
