package providers

import (
	"testing"

	"github.com/posilva/account-service/internal/core/domain"
	"github.com/stretchr/testify/require"
)

func TestProviderFactory_Get_ReturnsError_WhenProviderNotFound(t *testing.T) {
	factory := NewDefaultFactory()

	_, err := factory.Get("non-existent-provider")

	require.NotNil(t, err, "expected an error when provider is not found")
	require.ErrorIs(t, err, domain.ErrProviderNotFound, "expected ErrProviderNotFound error")

}
