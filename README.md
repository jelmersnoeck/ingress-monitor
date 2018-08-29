# Ingress Monitor

[![Build Status](https://travis-ci.org/jelmersnoeck/ingress-monitor.svg?branch=master)](https://travis-ci.org/jelmersnoeck/ingress-monitor)

Ingress Monitor is a Kubernetes Operator which takes care of setting up
monitoring for your Kubernetes Ingress and Service objects.

## Status

This is still a WIP. Currently resource syncs are not entirely implemented and
no provider mappings have been implemented.

## Installation

To install the Operator, make sure you have RBAC enabled in your cluster.

```
kubectl apply -f https://raw.githubusercontent.com/jelmersnoeck/ingress-monitor/master/docs/kube/with-rbac.yaml
```


## Supported Providers

Providers are used to indicate where we want to set up a monitor. Multiple
providers are supported within one cluster. These can be referenced later on in
a Monitor.

The Operator only supports a certain set of Monitoring Providers. Below we have
listed these providers.

### StatusCake

To configure StatusCake, there are 2 required arguments:

- username
- apiKey

As an optional argument, you can reference a `contactGroup` which will be used
to send notifications to.

All values follow the `EnvVar` schema, meaning you can use plaintext `values` or
`secretKeyRef`. We recommend using the `secretKeyRef`.

## Design

For more information about the design of this project, have a look at the
[design documents](./docs/design/README.md).
