// Package repository provides an adapter for DynamoDB output operations implements the AccountsRepository interface.
package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/posilva/account-service/internal/adapters/output/idgen"
	"github.com/posilva/account-service/internal/core/domain"
	"github.com/posilva/account-service/internal/core/ports"
)

// NOTE PMS: we could use the Table PK and SK to store the connection betwee the account and the provider
// still need to assess the pros and cons of this approach, but it seems to be a good fit for our use case
// for now we will use a GSI to store the connection between the account and the provider as it works

// Constants for DynamoDB table and index names
const (
	GSI1IndexName                  = "GSI1"
	GSI1PKName                     = "GSI1PK"
	GSI1SKName                     = "GSI1SK"
	AccountProviderPKPrefixFmt     = "ACNT#%s"
	AccountProviderSKPrefixFmt     = "PVDR#%s#%s"
	AccountProviderGSI1PKPrefixFmt = "PVDR#%s"
	AccountProviderGSI1SKPrefixFmt = "PVDRID#%s"
)

// DDBAccountProviderRecord represents the structure of an account provider record in DynamoDB.
// We use ISO8601 format for date strings to facilitate reading dates in DynamoDB, as this format also sorts correctly.
type DDBAccountProviderRecord struct {
	AccountID          string `dynamodbav:"AccountID"`
	ProviderType       string `dynamodbav:"ProviderType"`
	ProviderID         string `dynamodbav:"ProviderID"`
	DateCreatedISO8601 string `dynamodbav:"DateCreated"`
}

// DynamoDBAPI defines the interface for DynamoDB operations to make it easy to mock in tests as suggested in the docs
// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/unit-testing.html
// NOTE: We need to define here every SDK operation we want to use in our repository.
type DynamoDBAPI interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// dynamoDBAccountsRepository implements the AccountsRepository interface for DynamoDB.
type dynamoDBAccountsRepository struct {
	tableName   string
	idGenerator ports.IDGenerator
	client      DynamoDBAPI
}

// Safeguard check to ensure dynamoDBAccountsRepository implements the AccountsRepository interface
var _ ports.AccountsRepository = (*dynamoDBAccountsRepository)(nil)

// NewDynamoDBAccountsRepositoryWithIDGenerator creates a new instance of DynamoDBAccountsRepository with a custom ID generator.
func NewDynamoDBAccountsRepositoryWithIDGenerator(client DynamoDBAPI, tableName string, idGenerator ports.IDGenerator) ports.AccountsRepository {
	return &dynamoDBAccountsRepository{
		tableName:   tableName,
		idGenerator: idGenerator,
		client:      client,
	}
}

// NewDynamoDBAccountsRepository creates a new instance of DynamoDBAccountsRepository.
func NewDynamoDBAccountsRepository(client DynamoDBAPI, tableName string) ports.AccountsRepository {
	return NewDynamoDBAccountsRepositoryWithIDGenerator(client, tableName, idgen.NewKSUIDGenerator())
}

// ResolveIDByProvider resolves the account ID by provider type and provider ID.
// If the account does not exist, it returns an error indicating that the account was not found
func (r *dynamoDBAccountsRepository) ResolveIDByProvider(ctx context.Context, providerType domain.ProviderType, providerID string) (domain.AccountID, error) {
	// Resolve the account ID by provider type and provider ID using dynamoDB operations.
	// use go sdk v2 query builder to query the DynamoDB table

	gsk1pk := fmt.Sprintf(AccountProviderGSI1PKPrefixFmt, providerType)
	gsk1sk := fmt.Sprintf(AccountProviderGSI1SKPrefixFmt, providerID)

	expr, err := expression.NewBuilder().
		WithKeyCondition(expression.Key(GSI1PKName).Equal(expression.Value(gsk1pk)).
			And(expression.Key(GSI1SKName).Equal(expression.Value(gsk1sk)))).Build()

	if err != nil {
		return "", fmt.Errorf("failed to build expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
		IndexName:                 aws.String(GSI1IndexName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to query DynamoDB: %w", err)
	}
	if len(result.Items) == 0 {
		return domain.EmptyAccountID, domain.ErrAccountNotFound
	}

	if len(result.Items) > 1 {
		// in the future we may consider to just pick the first one, but for now we will return an error
		// as we cannot ensure the order of the items in the result this could lead to unexpected behavior
		// hard to debug
		return domain.EmptyAccountID, fmt.Errorf("unexpected multiple accounts found for provider type %s and provider ID %s", providerType, providerID)
	}

	record := &DDBAccountProviderRecord{}
	if err := attributevalue.UnmarshalMap(result.Items[0], record); err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
	}

	return domain.AccountID(record.AccountID), nil
}

// Create creates a new account in DynamoDB using the provider type and provider ID.
// It returns the newly created account ID or an error if the creation fails.
func (r *dynamoDBAccountsRepository) Create(ctx context.Context, providerType domain.ProviderType, providerID string) (domain.AccountID, error) {
	// Implementation for creating a new account in DynamoDB
	return domain.AccountID(r.idGenerator.GenerateID()), nil
}
