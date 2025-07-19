package ports

import (
	"context"

	"github.com/posilva/account-service/internal/core/domain"
)

// AuthService defines the interface for authentication services.
type AuthService interface {
	Authenticate(context.Context, domain.AuthenticateInput) (*domain.AuthenticateOutput, error)
}
