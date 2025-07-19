package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/posilva/account-service/internal/core/ports"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

// References:
// - https://pkg.go.dev/google.golang.org/api/androidpublisher/v3
// - https://developers.google.com/identity/sign-in/android/backend-auth
// - https://developer.android.com/games/pgs/sign-in

const (
	defaultTimeout = 2 * time.Second
)

type GoogleProvider struct {
	service *androidpublisher.Service
}

type googleAuthResult struct {
	ID string
}

func (r *googleAuthResult) GetID() string {
	return r.ID
}

// NewGoogleProvider creates a new GoogleProvider
// serviceAccount is a placeholder for the Google service account credentials in json format.
func NewGoogleProvider(serviceAccount string) (*GoogleProvider, error) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// TODO: this is not the right service for Google Play Authentication.
	service, err := androidpublisher.NewService(
		timeoutCtx,
		option.WithCredentialsJSON([]byte(serviceAccount)),
		option.WithScopes(androidpublisher.AndroidpublisherScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google service: %w", err)
	}

	return &GoogleProvider{
		service: service,
	}, nil
}

// Authenticate executes authentication with Google and returns an authresult.
func (p *GoogleProvider) Authenticate(ctx context.Context, data map[string]string) (ports.AuthResult, error) {
	return &googleAuthResult{ID: "dummy-google-id"}, nil
}
