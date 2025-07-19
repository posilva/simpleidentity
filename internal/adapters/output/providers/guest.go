package providers

import (
	"context"

	"github.com/posilva/account-service/internal/core/ports"
)

type GuestProvider struct{}

type guestAuthResult struct {
	ID string
}

func (r *guestAuthResult) GetID() string {
	return r.ID
}

func NewGuestProvider() *GuestProvider {
	return &GuestProvider{}
}

func (p *GuestProvider) Authenticate(ctx context.Context, data map[string]string) (ports.AuthResult, error) {
	return &guestAuthResult{
		ID: "guest-id",
	}, nil
}
