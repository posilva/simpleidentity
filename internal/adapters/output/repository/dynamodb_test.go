package repository

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	tableName := "accounts_test"

	ctrl := mock.NewMockController(t)

	clientMock := mock.Mock[DynamoDBAPI](ctrl)
	idGeneratorMock := mock.Mock[ports.IDGenerator](ctrl)

	mock.WhenSingle(idGeneratorMock.GenerateID()).ThenReturn(aid)
	mock.WhenDouble(clientMock.Query(mock.Any[context.Context](), mock.Any[*dynamodb.QueryInput]())).ThenAnswer(func(args []any) (*dynamodb.QueryOutput, error) {
		return &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{
					"AccountID":    &types.AttributeValueMemberS{Value: aid},
					"ProviderType": &types.AttributeValueMemberS{Value: string(providerType)},
					"ProviderID":   &types.AttributeValueMemberS{Value: providerID},
					"DateCreated":  &types.AttributeValueMemberS{Value: "2023-10-01T00:00:00Z"},
				},
			},
		}, nil
	})
	repo := NewDynamoDBAccountsRepositoryWithIDGenerator(clientMock, tableName, idGeneratorMock)
	accountID, err := repo.ResolveIDByProvider(ctx, providerType, providerID)

	require.NoError(t, err)
	require.NotEqual(t, accountID, domain.EmptyAccountID)
}

func TestDynamoDBAccountsRepository_CreateIdentity_ReturnsAccountID(t *testing.T) {
	ctx := context.Background()
	providerType := domain.ProviderTypeGuest
	providerID := "test_provider_id"
	aid := idgen.NewKSUIDGenerator().GenerateID()
	tableName := "accounts_test"

	ctrl := mock.NewMockController(t)

	clientMock := mock.Mock[DynamoDBAPI](ctrl)
	idGeneratorMock := mock.Mock[ports.IDGenerator](ctrl)

	mock.WhenSingle(idGeneratorMock.GenerateID()).ThenReturn(aid)

	repo := NewDynamoDBAccountsRepositoryWithIDGenerator(clientMock, tableName, idGeneratorMock)
	accountID, err := repo.Create(ctx, providerType, providerID)

	require.NotEqual(t, accountID, domain.EmptyAccountID)
	require.NoError(t, err)
}
