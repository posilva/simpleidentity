package providers

import (
	"context"

	"github.com/posilva/account-service/internal/core/ports"
)

type AppleProvider struct{}

type appleAuthResult struct {
	ID string
}

func NewAppleProvider() *AppleProvider {
	return &AppleProvider{}
}

func (r *appleAuthResult) GetID() string {
	return r.ID
}

func (p *AppleProvider) Authenticate(ctx context.Context, data map[string]string) (ports.AuthResult, error) {
	return &appleAuthResult{ID: "dummy-apple-id"}, nil
}
