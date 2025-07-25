package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/posilva/account-service/internal/adapters/output/idgen"
	"github.com/posilva/account-service/internal/adapters/output/repository"
	"github.com/posilva/account-service/internal/core/domain"
	"github.com/stretchr/testify/require"
)

// TestMain setup for Podman environment
func TestMain(m *testing.M) {
	// Setup Podman environment before running tests
	if IsPodmanAvailable() {
		SetupPodmanEnvironment()
		fmt.Println("Using Podman for testcontainers")
	} else {
		fmt.Println("Podman not found, falling back to Docker")
	}

	code := m.Run()
	os.Exit(code)
}

func TestAccountsRepositoryDynamodb_Integration(t *testing.T) {
	client, cleanup := setupDynamoDBContainer(t)
	defer cleanup()

	tableName := "users_test"
	createTestTable(t, client, tableName)

	repo := repository.NewDynamoDBAccountsRepository(client, tableName)
	ctx := context.Background()

	t.Run("ResolveIDByProvider returns ErrAccountNotFound", func(t *testing.T) {
		accountID, err := repo.ResolveIDByProvider(ctx, domain.ProviderTypeGuest, "test_provider_id")
		require.ErrorIs(t, err, domain.ErrAccountNotFound)
		require.Equal(t, domain.EmptyAccountID, accountID)
	})

	t.Run("Create account returns Successfully", func(t *testing.T) {
		accountID, err := repo.Create(ctx, domain.ProviderTypeGuest, "test_provider_id")
		require.Nil(t, err)
		require.NotEmpty(t, accountID)
	})

	t.Run("ResolveIDByProvider returns accountID", func(t *testing.T) {
		providerID := idgen.NewKSUIDGenerator().GenerateID()
		accountID, err := repo.Create(ctx, domain.ProviderTypeGuest, providerID)
		require.Nil(t, err)
		require.NotEmpty(t, accountID)

		resolvedAccountID, err := repo.ResolveIDByProvider(ctx, domain.ProviderTypeGuest, providerID)
		require.Nil(t, err)
		require.Equal(t, resolvedAccountID, accountID)
	})

	t.Run("Create account returns Provider ID already exists", func(t *testing.T) {
		providerID := idgen.NewKSUIDGenerator().GenerateID()
		accountID, err := repo.Create(ctx, domain.ProviderTypeGuest, providerID)
		require.Nil(t, err)
		require.NotEmpty(t, accountID)
		empty, err := repo.Create(ctx, domain.ProviderTypeGuest, providerID)
		require.ErrorIs(t, err, domain.ErrProviderIDOrAccountAlreadyExists)
		require.Equal(t, domain.EmptyAccountID, empty)
	})
}
