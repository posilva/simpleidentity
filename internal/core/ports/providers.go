package ports

import (
	"context"

	"github.com/posilva/account-service/internal/core/domain"
)

// AuthResult defines the interface for providers authentication results.
type AuthResult interface {
	GetID() string
}

// AuthProvider defines the interface for authentication providers.
type AuthProvider interface {
	Authenticate(context.Context, map[string]string) (AuthResult, error)
}

// AuthProviderFactory defines the interface for creating authentication providers.
type AuthProviderFactory interface {
	Get(providerType domain.ProviderType) (AuthProvider, error)
	Add(providerType domain.ProviderType, provider AuthProvider) error
	Remove(providerType domain.ProviderType) error
}
