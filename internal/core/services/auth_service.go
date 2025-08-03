package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/posilva/simpleidentity/internal/core/domain"
	"github.com/posilva/simpleidentity/internal/core/ports"
)

// AuthService is the implementation of the AuthService interface.
type authService struct {
	providerFactory ports.AuthProviderFactory
	repository      ports.AccountsRepository
}

// Safegard check to ensure authService implements the AuthService interface
var _ ports.AuthService = (*authService)(nil)

// NewAuthService creates a new instance of AuthService with the given provider factory.
func NewAuthService(providerFactory ports.AuthProviderFactory, r ports.AccountsRepository) *authService {
	return &authService{
		providerFactory: providerFactory,
		repository:      r,
	}
}

// Authenticate authenticates a user using the specified authentication provider.
func (s *authService) Authenticate(ctx context.Context, input domain.AuthenticateInput) (*domain.AuthenticateOutput, error) {
	provider, err := s.providerFactory.Get(input.ProviderType)
	if err != nil {
		return nil, err
	}

	result, err := provider.Authenticate(ctx, input.AuthData)
	if err != nil {
		return nil, err
	}
	accountID, err := s.repository.ResolveIDByProvider(ctx, input.ProviderType, result.GetID())
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			// this means that the account does not exist, so we need to create it
			accountID, err := s.repository.Create(ctx, input.ProviderType, result.GetID())
			if err != nil {
				return nil, fmt.Errorf("failed to create account: %w", err)
			}
			return &domain.AuthenticateOutput{
				AccountID: accountID,
				IsNew:     true,
			}, nil
		}
		return nil, fmt.Errorf("failed to resolve account ID: %w", err)
	}

	return &domain.AuthenticateOutput{
		AccountID: accountID,
	}, nil
}
