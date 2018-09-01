package provider

import (
	"errors"
	"sync"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
)

var defaultProviderFactory = newFactory()

// ErrProviderNotFound is an error which is used when we try to create a new
// client for a given provider which isn't registered with the factory.
var ErrProviderNotFound = errors.New("the specified provider can't be found")

// FactoryFunc is the interface used to allow creating a new provider. This
// shoud be used by provider wrappers to allow for creating new clients.
type FactoryFunc func(v1alpha1.NamespacedProvider) (Interface, error)

// FactoryInterface is the interface used for a ProviderFactory. It allows you
// to fetch providers from a local store and use them to configure monitors.
type FactoryInterface interface {
	Register(string, FactoryFunc)
	From(v1alpha1.NamespacedProvider) (Interface, error)
}

// Register registers a provider which can be used from within the Factory to
// create new Monitors.
func Register(name string, ff FactoryFunc) {
	defaultProviderFactory.Register(name, ff)
}

// From returns a provider from the given NamespacedProvider.
func From(prov v1alpha1.NamespacedProvider) (Interface, error) {
	return defaultProviderFactory.From(prov)
}

// SimpleFactory is a factory object that knows how to get providers.
type SimpleFactory struct {
	providers map[string]FactoryFunc
	lock      sync.RWMutex
}

// Register registers the given provider with the factory under the given name.
func (pf *SimpleFactory) Register(name string, ff FactoryFunc) {
	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.providers[name] = ff
}

// From creates a new provider from the given configuration. This can then be
// used to register the provider within the
func (pf *SimpleFactory) From(prov v1alpha1.NamespacedProvider) (Interface, error) {
	pf.lock.RLock()
	defer pf.lock.RUnlock()

	pr, ok := pf.providers[prov.Type]
	if !ok {
		return nil, ErrProviderNotFound
	}

	return pr(prov)
}

// Flush resets all FactoryFuncs
func (pf *SimpleFactory) Flush() {
	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.providers = map[string]FactoryFunc{}
}

func newFactory() *SimpleFactory {
	return &SimpleFactory{
		providers: map[string]FactoryFunc{},
	}
}

// DefaultFactory returns the DefaultFactory.
func DefaultFactory() *SimpleFactory {
	return defaultProviderFactory
}
