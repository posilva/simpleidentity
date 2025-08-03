// Package providers provides authentication providers
package providers

import (
	"github.com/posilva/simpleidentity/internal/core/domain"
	"github.com/posilva/simpleidentity/internal/core/ports"
)

type defaultFactory struct {
	registry map[domain.ProviderType]ports.AuthProvider
}

func NewDefaultFactory() ports.AuthProviderFactory {
	return &defaultFactory{
		registry: make(map[domain.ProviderType]ports.AuthProvider),
	}
}

func (d *defaultFactory) Add(providerType domain.ProviderType, provider ports.AuthProvider) error {
	d.registry[providerType] = provider
	return nil
}

func (d *defaultFactory) Get(providerType domain.ProviderType) (ports.AuthProvider, error) {
	if provider, exists := d.registry[providerType]; exists {
		return provider, nil
	}
	return nil, domain.ErrProviderNotFound
}

func (d *defaultFactory) Remove(providerType domain.ProviderType) error {
	delete(d.registry, providerType)
	return nil
}
