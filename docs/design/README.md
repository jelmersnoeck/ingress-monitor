# Ingress Monitor Design

The Ingress Monitor Custom Resources are designed with reusability and
resilience in mind. The goal is to create a set of reusable components which can
be used to apply configurations to multiple Ingresses at once and allowing
multiple providers or provider groups to be configured.

## Provider

The Provider Resource allows administrators to configure different type of
providers. This is useful for when you want to monitor different kind of
applications which each have their own alert group.

For example, for backend applications, you want to alert your backend team. For
frontend applications, you want to alert your frontend team.

For more information, see the [Provider documentation](./provider.md)

## Monitor

A Monitor is a high level configuration to set up checks for your Ingresses. A
Monitor can be used to cover multiple Ingresses.

For more information, see the [Monitor documentation](./monitor.md)

## IngressMonitor

IngressMonitors are fully configured monitors linked to an Ingress and Provider.

The Operator reacts on IngressMonitor events to either Create, Update or Delete
a Monitor with the configured Provider.

IngressMonitors are automatically configured by adding Monitors and Ingresses
to your cluster, but can be used to configure an external check as well.
