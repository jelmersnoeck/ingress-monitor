# Monitor

A Monitor is used to set up a high level configuration for a set of checks. It
can be used to configure multiple Ingresses by setting up a selector.

For each Ingress that matches the selector, the Operator creates an
[IngressMonitor](./ingress-monitor.md).
