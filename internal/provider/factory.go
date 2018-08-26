package provider

import "sync"

var defaultProviderFactory = newProviderFactory()

// RegisterProvider registers a provider which can be used from within the
// Factory to create new Monitors.
func RegisterProvider(name string, provider Interface) {
	defaultProviderFactory.Register(name, provider)
}

// ProviderFor returns the provider for a given name.
func ProviderFor(name string) (Interface, bool) {
	return defaultProviderFactory.ProviderFor(name)
}

// DeregisterProvider removes a provider from the registry.
func DeregisterProvider(name string) {
	defaultProviderFactory.Deregister(name)
}

// ProviderFactory is a factory object that knows how to get providers.
type ProviderFactory struct {
	providers map[string]Interface
	lock      sync.RWMutex
}

// Register registers the given provider with the factory under the given name.
func (pf *ProviderFactory) Register(name string, provider Interface) {
	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.providers[name] = provider
}

// ProviderFor gets the registered provider for the given name.
func (pf *ProviderFactory) ProviderFor(name string) (Interface, bool) {
	pf.lock.RLock()
	defer pf.lock.RUnlock()

	pr, ok := pf.providers[name]
	return pr, ok
}

// Deregister deregisters the provider with the given name.
func (pf *ProviderFactory) Deregister(name string) {
	pf.lock.Lock()
	defer pf.lock.Unlock()

	delete(pf.providers, name)
}

func newProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: map[string]Interface{},
		lock:      sync.RWMutex{},
	}
}
