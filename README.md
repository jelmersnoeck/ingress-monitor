# Ingress Monitor

[![Build Status](https://travis-ci.org/jelmersnoeck/ingress-monitor.svg?branch=master)](https://travis-ci.org/jelmersnoeck/ingress-monitor)

Ingress Monitor is a Kubernetes Operator which takes care of setting up
monitoring for your Kubernetes Ingress and Service objects.

## Status

This is still a WIP. The main piece currently missing is Custom Resource
Definition validation and extra providers.

## Installation

To install the Operator, make sure you have RBAC enabled in your cluster.

```
kubectl apply -f https://raw.githubusercontent.com/jelmersnoeck/ingress-monitor/master/docs/kube/with-rbac.yaml
```

## Example

There is an example installed in [the examples directory](./_examples/kuard). This is using
StatusCake as a provider and is using [kuard](https://github.com/kubernetes-up-and-running/kuard) as the application it monitors.

This is meant to demonstrate the configuration options.

To use this, first you'll need to create an account with StatusCake and retrieve
the API key.

Following that, you can set up a secret which contains your credentials:

```
kubectl create namespace websites
kubectl create secret generic statuscake-secrets -n websites --from-literal=username=<STATUSCAKE_USERNAME> --from-literal=apikey=<STATUSCAKE_APIKEY>
```

After this, you can apply the entire `_examples/kuard` folder:

```
kubectl apply -f _examples/kuard
```

This will set up a Provider, MonitorTemplate and Monitor along with a Deployment
which is exposed through a Service and Ingress. The Ingress has the label
`team: gophers` which is used by the Monitor to create an IngressMonitor.

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
