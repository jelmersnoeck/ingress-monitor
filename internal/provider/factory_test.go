package provider_test

import (
	"reflect"
	"testing"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/fake"
)

func TestProviderFactory(t *testing.T) {
	fact := provider.NewFactory(nil)
	reset := func() {
		fact = provider.NewFactory(nil)
	}

	t.Run("with registered provider", func(t *testing.T) {
		defer reset()

		prov := new(fake.SimpleProvider)

		fact.Register("simple", fake.FactoryFunc(prov))

		cl, err := fact.From(v1alpha1.NamespacedProvider{
			ProviderSpec: v1alpha1.ProviderSpec{
				Type: "simple",
			},
		})

		if err != nil {
			t.Fatalf("Expected no error getting the provider, got: %s", err)
		}

		if !reflect.DeepEqual(cl, prov) {
			t.Errorf("Expected new client to be the test client")
		}
	})

	t.Run("without registered provider", func(t *testing.T) {
		defer reset()

		_, err := fact.From(v1alpha1.NamespacedProvider{
			ProviderSpec: v1alpha1.ProviderSpec{
				Type: "simple",
			},
		})

		if err != provider.ErrProviderNotFound {
			t.Errorf("Expected error `%s`, got `%s`", provider.ErrProviderNotFound, err)
		}
	})
}
