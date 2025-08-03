// Package repository provides an adapter for DynamoDB output operations implements the AccountsRepository interface.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/posilva/simpleidentity/internal/adapters/output/idgen"
	"github.com/posilva/simpleidentity/internal/core/domain"
	"github.com/posilva/simpleidentity/internal/core/ports"
)

// NOTE PMS: we could use the Table PK and SK to store the connection betwee the account and the provider
// still need to assess the pros and cons of this approach, but it seems to be a good fit for our use case
// for now we will use a GSI to store the connection between the account and the provider as it works

// Constants for DynamoDB table and index names
const (
	TablePKName                = "PK"
	TableSKName                = "SK"
	AccountIdentitySKName      = "IDENTITY"
	AccountProviderPKPrefixFmt = "ACNT#%s"
	AccountProviderSKPrefixFmt = "PVDR#%s#%s"
)

// errTransactionErrorConditionFailed is an internal error
var errTransactionErrorConditionFailed = errors.New("transaction error ConditionalCheckFailed")

// DDBAccountProviderRecordData represents the data of an account provider record in DynamoDB.
// We use ISO8601 format for date strings to facilitate reading dates in DynamoDB, as this format also sorts correctly.
type DDBAccountProviderRecordData struct {
	AccountID          string `dynamodbav:"AccountID"`
	ProviderType       string `dynamodbav:"ProviderType"`
	ProviderID         string `dynamodbav:"ProviderID"`
	DateCreatedISO8601 string `dynamodbav:"DateCreated"`
}

// DDBAccountProviderRecord represents an account provider record in DynamoDB with primary key of the table and GSI
type DDBAccountProviderRecord struct {
	DDBAccountProviderRecordData
	PK string `dynamodbav:"PK"`
	SK string `dynamodbav:"SK"`
}

// DynamoDBAPI defines the interface for DynamoDB operations to make it easy to mock in tests as suggested in the docs
// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/unit-testing.html
// NOTE: We need to define here every SDK operation we want to use in our repository.
type DynamoDBAPI interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
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

	pk := fmt.Sprintf(AccountProviderSKPrefixFmt, providerType, providerID)
	sk := AccountIdentitySKName
	pkExp := expression.Key(TablePKName).Equal(expression.Value(pk))
	skExp := expression.Key(TableSKName).Equal(expression.Value(sk))

	expr, err := expression.NewBuilder().WithKeyCondition(pkExp.And(skExp)).Build()
	if err != nil {
		return "", fmt.Errorf("failed to build expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
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

	record := &DDBAccountProviderRecordData{}
	if err := attributevalue.UnmarshalMap(result.Items[0], record); err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
	}

	return domain.AccountID(record.AccountID), nil
}

// Create creates a new account in DynamoDB using the provider type and provider ID.
// It returns the newly created account ID or an error if the creation fails.
func (r *dynamoDBAccountsRepository) Create(ctx context.Context, providerType domain.ProviderType, providerID string) (domain.AccountID, error) {
	accountID := r.idGenerator.GenerateID()

	identityCond := expression.And(
		expression.AttributeNotExists(expression.Name(TablePKName)),
		expression.AttributeNotExists(expression.Name(TableSKName)),
	)

	data := DDBAccountProviderRecordData{
		AccountID:          accountID,
		ProviderType:       string(providerType),
		ProviderID:         providerID,
		DateCreatedISO8601: time.Now().UTC().Format(time.RFC3339),
	}

	identityRecord := DDBAccountProviderRecord{
		PK:                           fmt.Sprintf(AccountProviderSKPrefixFmt, providerType, providerID),
		SK:                           AccountIdentitySKName,
		DDBAccountProviderRecordData: data,
	}
	identityExpr, err := expression.NewBuilder().
		WithCondition(identityCond).
		Build()
	if err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to build identity expression: %w", err)
	}

	identityItem, err := attributevalue.MarshalMap(identityRecord)
	if err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to marshal identity record: %w", err)
	}

	accountCond := expression.And(
		expression.AttributeNotExists(expression.Name(TablePKName)),
		expression.AttributeNotExists(expression.Name(TableSKName)),
	)

	accountExpr, err := expression.NewBuilder().WithCondition(accountCond).Build()
	if err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to build account expression: %w", err)
	}

	accountRecord := DDBAccountProviderRecord{
		PK:                           fmt.Sprintf(AccountProviderPKPrefixFmt, accountID),
		SK:                           fmt.Sprintf(AccountProviderSKPrefixFmt, providerType, providerID),
		DDBAccountProviderRecordData: data,
	}

	accountItem, err := attributevalue.MarshalMap(accountRecord)
	if err != nil {
		return domain.EmptyAccountID, fmt.Errorf("failed to marshal account record: %w", err)
	}
	input := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName:                 aws.String(r.tableName),
					Item:                      identityItem,
					ConditionExpression:       identityExpr.Condition(),
					ExpressionAttributeNames:  identityExpr.Names(),
					ExpressionAttributeValues: identityExpr.Values(),
				},
			},
			{
				Put: &types.Put{
					TableName:                 aws.String(r.tableName),
					Item:                      accountItem,
					ConditionExpression:       accountExpr.Condition(),
					ExpressionAttributeNames:  accountExpr.Names(),
					ExpressionAttributeValues: accountExpr.Values(),
				},
			},
		},
	}

	_, err = r.client.TransactWriteItems(ctx, input)
	if err != nil {
		tErr := enrichErrorWithOperationContext(err, []string{"PUT Provider Identity data", "PUT Account data"})
		if errors.Is(tErr, errTransactionErrorConditionFailed) {
			tErr = domain.ErrProviderIDOrAccountAlreadyExists
		}
		return domain.EmptyAccountID, fmt.Errorf("failed to execute transaction when creating account: %w", tErr)
	}

	return domain.AccountID(accountID), nil
}

// enrichErrorWithOperationContext extracts transaction related error from the SDK error
func enrichErrorWithOperationContext(err error, operations []string) error {
	var transactionCancelledErr *types.TransactionCanceledException
	if errors.As(err, &transactionCancelledErr) {
		if transactionCancelledErr.CancellationReasons != nil {
			for i, reason := range transactionCancelledErr.CancellationReasons {
				if reason.Code != nil && *reason.Code != "None" {
					operationName := "Unknown"
					if i < len(operations) {
						operationName = operations[i]
					}

					reasonStr := *reason.Code
					if reason.Message != nil {
						reasonStr += ": " + *reason.Message
					}

					// custom sentinel errors to allow to bubble up the error with a specific semantic
					err = fmt.Errorf("transaction error %s", *reason.Code)
					if *reason.Code == "ConditionalCheckFailed" {
						err = errTransactionErrorConditionFailed
					}
					return fmt.Errorf("operation: %s, index: %d, reason: %s: %w",
						operationName, i, reasonStr, err)
				}
			}
		}
	}

	return err
}

func ListWrappedErrors(err error) []error {
	var chain []error
	for err != nil {
		chain = append(chain, err)
		err = errors.Unwrap(err)
	}
	return chain
}
