package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/posilva/account-service/internal/adapters/output/repository"
	"github.com/posilva/account-service/internal/core/domain"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Utility function to check if Podman is available
func IsPodmanAvailable() bool {
	_, err := os.Stat("/usr/bin/podman")
	if err == nil {
		return true
	}

	// Check in other common locations
	locations := []string{
		"/usr/local/bin/podman",
		"/opt/homebrew/bin/podman",
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return true
		}
	}

	return false
}

// SetupPodmanEnvironment configures environment variables for Podman
func SetupPodmanEnvironment() {
	// Method 1: Use Podman socket directly
	if _, exists := os.LookupEnv("DOCKER_HOST"); !exists {
		// For rootless Podman
		if uid := os.Getuid(); uid != 0 {
			podmanSocket := fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid)
			os.Setenv("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", uid))
			os.Setenv("DOCKER_HOST", podmanSocket)
		} else {
			// For rootful Podman
			os.Setenv("DOCKER_HOST", "unix:///run/podman/podman.sock")
		}
	}

	// Disable TLS for local Podman socket
	os.Setenv("DOCKER_TLS_VERIFY", "")
	os.Setenv("DOCKER_CERT_PATH", "")

	// Use Podman API version
	os.Setenv("DOCKER_API_VERSION", "1.40")

	// Testcontainers specific settings
	os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/run/user/"+fmt.Sprintf("%d", os.Getuid())+"/podman/podman.sock")
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true") // Disable ryuk as it may not work well with Podman
}

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

func setupDynamoDBContainer(t *testing.T) (*dynamodb.Client, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "amazon/dynamodb-local:3.0.0",
		ExposedPorts: []string{"8000/tcp"},
		Cmd:          []string{"-jar", "DynamoDBLocal.jar", "-inMemory", "-sharedDb"},
		WaitingFor:   wait.ForListeningPort("8000/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get container endpoint
	host, err := container.Host(ctx)
	require.NoError(t, err)
	fmt.Println("DynamoDB Local is running at:", host)

	port, err := container.MappedPort(ctx, "8000")
	require.NoError(t, err)
	fmt.Println("DynamoDB Local port:", port.Port())

	endpoint := "http://" + host + ":" + port.Port()
	fmt.Println("DynamoDB Local endpoint:", endpoint)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: region,
				}, nil
			}),
		),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SessionToken:    "",
			},
		}),
	)
	require.NoError(t, err)

	client := dynamodb.NewFromConfig(cfg)

	// Cleanup function
	cleanup := func() {
		container.Terminate(ctx)
	}

	return client, cleanup
}

func createTestTable(t *testing.T, client *dynamodb.Client, tableName string) {
	ctx := context.Background()

	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: &tableName,
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("SK"),
				KeyType:       types.KeyTypeRange,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI1PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI1SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("GSI1PK"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("GSI1SK"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll, // or ProjectionTypeKeysOnly, ProjectionTypeInclude
				},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	log.Println("Waiting for table to be active...")
	// Wait for table to be active
	waiter := dynamodb.NewTableExistsWaiter(client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: &tableName,
	}, 150*time.Second)

	require.NoError(t, err)

	out, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: &tableName,
	})
	log.Println("checking created table:", tableName)
	require.NoError(t, err)
	require.NotNil(t, out.Table)
	require.Equal(t, tableName, *out.Table.TableName)
	require.Equal(t, types.TableStatusActive, out.Table.TableStatus)
	log.Println("Table created successfully:", out.Table.CreationDateTime.Format(time.RFC3339))
}

func TestAccountsRepositoryDynamodb_Integration(t *testing.T) {
	client, cleanup := setupDynamoDBContainer(t)
	defer cleanup()

	tableName := "users_test"
	createTestTable(t, client, tableName)

	repo := repository.NewDynamoDBAccountsRepository(client, tableName)
	ctx := context.Background()

	t.Run("ResolveIDByProvider returns ErrAccountNotFound", func(t *testing.T) {
		accountId, err := repo.ResolveIDByProvider(ctx, domain.ProviderTypeGuest, "test_provider_id") // Example usage
		require.ErrorIs(t, err, domain.ErrAccountNotFound)
		require.Equal(t, domain.EmptyAccountID, accountId)
	})
}
