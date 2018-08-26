package provider_test

import (
	"reflect"
	"testing"

	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/fake"
)

func TestProviderFactory(t *testing.T) {
	testProvider := &fake.SimpleProvider{}

	provider.RegisterProvider("simple", testProvider)

	pr, ok := provider.ProviderFor("simple")
	if !ok {
		t.Fatalf("Expected a provider to be registered, got none")
	}

	if !reflect.DeepEqual(pr, testProvider) {
		t.Errorf("Expected the fetched provider to equal the registered provider")
	}

	newProvider := &fake.SimpleProvider{
		CreateFunc: func(provider.IngressMonitor) error {
			return nil
		},
	}
	provider.RegisterProvider("simple", newProvider)

	pr, ok = provider.ProviderFor("simple")
	if !ok {
		t.Fatalf("Expected a provider to be registered, got none")
	}

	if reflect.DeepEqual(pr, testProvider) || !reflect.DeepEqual(pr, newProvider) {
		t.Errorf("Expected the fetched provider to equal the newProvider, not the testProvider")
	}

	provider.DeregisterProvider("simple")

	if _, ok = provider.ProviderFor("simple"); ok {
		t.Fatalf("Expected no provider to be registered, got one")
	}
}
