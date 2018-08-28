package provider

import "sync"

var defaultProviderFactory = newFactory()

// FactoryInterface is the interface used for a ProviderFactory. It allows you
// to fetch providers from a local store and use them to configure monitors.
type FactoryInterface interface {
	Deregister(string)
	Get(string) (Interface, bool)
	Register(string, Interface)
}

// RegisterProvider registers a provider which can be used from within the
// Factory to create new Monitors.
func RegisterProvider(name string, provider Interface) {
	defaultProviderFactory.Register(name, provider)
}

// Get returns the provider for a given name.
func Get(name string) (Interface, bool) {
	return defaultProviderFactory.Get(name)
}

// DeregisterProvider removes a provider from the registry.
func DeregisterProvider(name string) {
	defaultProviderFactory.Deregister(name)
}

// SimpleFactory is a factory object that knows how to get providers.
type SimpleFactory struct {
	providers map[string]Interface
	lock      sync.RWMutex
}

// Register registers the given provider with the factory under the given name.
func (pf *SimpleFactory) Register(name string, provider Interface) {
	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.providers[name] = provider
}

// Get gets the registered provider for the given name.
func (pf *SimpleFactory) Get(name string) (Interface, bool) {
	pf.lock.RLock()
	defer pf.lock.RUnlock()

	pr, ok := pf.providers[name]
	return pr, ok
}

// Deregister deregisters the provider with the given name.
func (pf *SimpleFactory) Deregister(name string) {
	pf.lock.Lock()
	defer pf.lock.Unlock()

	delete(pf.providers, name)
}

func newFactory() *SimpleFactory {
	return &SimpleFactory{
		providers: map[string]Interface{},
		lock:      sync.RWMutex{},
	}
}

// DefaultFactory returns the DefaultFactory.
func DefaultFactory() *SimpleFactory {
	return defaultProviderFactory
}
