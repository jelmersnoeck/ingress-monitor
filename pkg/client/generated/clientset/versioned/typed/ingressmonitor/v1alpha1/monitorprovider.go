// MIT License
//
// Copyright (c) 2018 Jelmer Snoeck
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	scheme "github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MonitorProvidersGetter has a method to return a MonitorProviderInterface.
// A group's client should implement this interface.
type MonitorProvidersGetter interface {
	MonitorProviders(namespace string) MonitorProviderInterface
}

// MonitorProviderInterface has methods to work with MonitorProvider resources.
type MonitorProviderInterface interface {
	Create(*v1alpha1.MonitorProvider) (*v1alpha1.MonitorProvider, error)
	Update(*v1alpha1.MonitorProvider) (*v1alpha1.MonitorProvider, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.MonitorProvider, error)
	List(opts v1.ListOptions) (*v1alpha1.MonitorProviderList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MonitorProvider, err error)
	MonitorProviderExpansion
}

// monitorProviders implements MonitorProviderInterface
type monitorProviders struct {
	client rest.Interface
	ns     string
}

// newMonitorProviders returns a MonitorProviders
func newMonitorProviders(c *IngressmonitorV1alpha1Client, namespace string) *monitorProviders {
	return &monitorProviders{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the monitorProvider, and returns the corresponding monitorProvider object, and an error if there is any.
func (c *monitorProviders) Get(name string, options v1.GetOptions) (result *v1alpha1.MonitorProvider, err error) {
	result = &v1alpha1.MonitorProvider{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("monitorproviders").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MonitorProviders that match those selectors.
func (c *monitorProviders) List(opts v1.ListOptions) (result *v1alpha1.MonitorProviderList, err error) {
	result = &v1alpha1.MonitorProviderList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("monitorproviders").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested monitorProviders.
func (c *monitorProviders) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("monitorproviders").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a monitorProvider and creates it.  Returns the server's representation of the monitorProvider, and an error, if there is any.
func (c *monitorProviders) Create(monitorProvider *v1alpha1.MonitorProvider) (result *v1alpha1.MonitorProvider, err error) {
	result = &v1alpha1.MonitorProvider{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("monitorproviders").
		Body(monitorProvider).
		Do().
		Into(result)
	return
}

// Update takes the representation of a monitorProvider and updates it. Returns the server's representation of the monitorProvider, and an error, if there is any.
func (c *monitorProviders) Update(monitorProvider *v1alpha1.MonitorProvider) (result *v1alpha1.MonitorProvider, err error) {
	result = &v1alpha1.MonitorProvider{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("monitorproviders").
		Name(monitorProvider.Name).
		Body(monitorProvider).
		Do().
		Into(result)
	return
}

// Delete takes name of the monitorProvider and deletes it. Returns an error if one occurs.
func (c *monitorProviders) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("monitorproviders").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *monitorProviders) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("monitorproviders").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched monitorProvider.
func (c *monitorProviders) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MonitorProvider, err error) {
	result = &v1alpha1.MonitorProvider{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("monitorproviders").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}