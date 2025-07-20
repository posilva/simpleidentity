package dynamodb

import (
	"context"
	"testing"

	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/posilva/account-service/internal/adapters/output/idgen"
	"github.com/posilva/account-service/internal/core/domain"
	"github.com/posilva/account-service/internal/core/ports"
	"github.com/stretchr/testify/require"
)

func TestDynamoDBAccountsRepository_ResolveIDByProvider_ReturnsAccountID(t *testing.T) {
	ctx := context.Background()
	providerType := domain.ProviderTypeGuest
	providerID := "test_provider_id"
	aid := idgen.NewKSUIDGenerator().GenerateID()

	ctrl := mock.NewMockController(t)

	clientMock := mock.Mock[DynamoDBAPI](ctrl)
	idGeneratorMock := mock.Mock[ports.IDGenerator](ctrl)

	mock.WhenSingle(idGeneratorMock.GenerateID()).ThenReturn(aid)

	repo := NewDynamoDBAccountsRepositoryWithIDGenerator(idGeneratorMock, clientMock)
	accountID, err := repo.ResolveIDByProvider(ctx, providerType, providerID)

	require.NoError(t, err)
	require.NotEmpty(t, accountID)
}
