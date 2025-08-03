// Package ports contains interfaces for core domain entities.
package ports

import (
	"context"

	"github.com/posilva/simpleidentity/internal/core/domain"
)

// AuthService defines the interface for authentication services.
type AuthService interface {
	Authenticate(context.Context, domain.AuthenticateInput) (*domain.AuthenticateOutput, error)
}

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

// AccountsRepository defines the interface for account repository operations.
type AccountsRepository interface {
	ResolveIDByProvider(context.Context, domain.ProviderType, string) (domain.AccountID, error)
	Create(context.Context, domain.ProviderType, string) (domain.AccountID, error)
}

// IDGenerator defines the interface for generating unique account IDs.
type IDGenerator interface {
	GenerateID() string
}
