package providers

import (
	"testing"

	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/posilva/account-service/internal/core/domain"
	"github.com/posilva/account-service/internal/core/ports"
	"github.com/stretchr/testify/require"
)

func TestProviderFactory_Get_ReturnsError_WhenProviderNotFound(t *testing.T) {
	factory := NewDefaultFactory()

	_, err := factory.Get("non-existent-provider")

	require.NotNil(t, err, "expected an error when provider is not found")
	require.ErrorIs(t, err, domain.ErrProviderNotFound, "expected ErrProviderNotFound error")
}

func TestProviderFactory_AddGetAndRemove_ReturnsProvider(t *testing.T) {
	ctrl := mock.NewMockController(t)
	authProviderMock := mock.Mock[ports.AuthProvider](ctrl)

	factory := NewDefaultFactory()
	err := factory.Add(domain.ProviderTypeGuest, authProviderMock)
	require.NoError(t, err)

	provider, err := factory.Get(domain.ProviderTypeGuest)
	require.NoError(t, err)
	require.Equal(t, authProviderMock, provider)

	err = factory.Remove(domain.ProviderTypeGuest)
	require.NoError(t, err)

	_, err = factory.Get(domain.ProviderTypeGuest)
	require.NotNil(t, err, "expected an error when provider is not found")
	require.ErrorIs(t, err, domain.ErrProviderNotFound, "expected ErrProviderNotFound error")
}
