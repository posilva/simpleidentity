package services

import (
	"context"
	"testing"

	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/posilva/account-service/internal/core/domain"
	"github.com/posilva/account-service/internal/core/ports"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
)

func TestAuthService_New_ReturnsANewInstance(t *testing.T) {
	ctrl := mock.NewMockController(t)
	factoryMock := mock.Mock[ports.AuthProviderFactory](ctrl)
	repoMock := mock.Mock[ports.AccountsRepository](ctrl)
	authService := NewAuthService(factoryMock, repoMock)
	require.NotNil(t, authService)
}

func TestAuthService_AuthenticateGuest_ReturnsAuthenticateOutputOK(t *testing.T) {
	// setup data
	authData := map[string]string{"id": "some_client_generated_id"}
	uid := ksuid.New().String()
	providerType := domain.ProviderTypeGuest
	// setup mocks
	ctrl := mock.NewMockController(t)
	factoryMock := mock.Mock[ports.AuthProviderFactory](ctrl)
	repoMock := mock.Mock[ports.AccountsRepository](ctrl)
	providerMock := mock.Mock[ports.AuthProvider](ctrl)
	authResultMock := mock.Mock[ports.AuthResult](ctrl)
	ctx := context.Background()
	// setup expectations
	mock.WhenSingle(authResultMock.GetID()).ThenReturn(uid)
	mock.WhenDouble(providerMock.Authenticate(ctx, authData)).ThenReturn(authResultMock, nil)
	mock.WhenDouble(factoryMock.Get(providerType)).ThenReturn(providerMock, nil)
	mock.WhenDouble(repoMock.ResolveIDByProvider(ctx, providerType, uid)).ThenReturn(domain.AccountID(uid), nil)
	// create the AuthService instance
	authService := NewAuthService(factoryMock, repoMock)
	output, err := authService.Authenticate(ctx, domain.AuthenticateInput{
		ProviderType: providerType,
		AuthData:     authData,
	})

	// assertions
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, domain.AccountID(uid), output.AccountID)
}

func TestAuthService_AuthenticateGuest_ReturnsErrorProviderNotFound(t *testing.T) {
	// setup data
	authData := map[string]string{"id": "some_client_generated_id"}
	providerType := domain.ProviderTypeGuest
	// setup mocks
	ctrl := mock.NewMockController(t)
	factoryMock := mock.Mock[ports.AuthProviderFactory](ctrl)
	repoMock := mock.Mock[ports.AccountsRepository](ctrl)
	ctx := context.Background()
	// setup expectations
	mock.WhenDouble(factoryMock.Get(providerType)).ThenReturn(nil, domain.ErrProviderNotFound)
	// create the AuthService instance
	authService := NewAuthService(factoryMock, repoMock)
	output, err := authService.Authenticate(ctx, domain.AuthenticateInput{
		ProviderType: providerType,
		AuthData:     authData,
	})
	// assertions
	require.Error(t, err)
	require.Nil(t, output)
	require.Equal(t, domain.ErrProviderNotFound, err)
}

func TestAuthService_AuthenticateGuest_ReturnAccountIsNew(t *testing.T) {
	// setup data
	authData := map[string]string{"id": "some_client_generated_id"}
	uid := ksuid.New().String()
	providerType := domain.ProviderTypeGuest
	// setup mocks
	ctrl := mock.NewMockController(t)
	factoryMock := mock.Mock[ports.AuthProviderFactory](ctrl)
	repoMock := mock.Mock[ports.AccountsRepository](ctrl)
	providerMock := mock.Mock[ports.AuthProvider](ctrl)
	authResultMock := mock.Mock[ports.AuthResult](ctrl)
	ctx := context.Background()
	// setup expectations
	mock.WhenSingle(authResultMock.GetID()).ThenReturn(uid)
	mock.WhenDouble(providerMock.Authenticate(ctx, authData)).ThenReturn(authResultMock, nil)
	mock.WhenDouble(factoryMock.Get(providerType)).ThenReturn(providerMock, nil)
	mock.WhenDouble(repoMock.ResolveIDByProvider(ctx, providerType, uid)).ThenReturn(domain.AccountID(""), domain.ErrAccountNotFound)
	mock.WhenDouble(repoMock.Create(ctx, providerType, uid)).ThenReturn(domain.AccountID(uid), nil)
	// create the AuthService instance
	authService := NewAuthService(factoryMock, repoMock)
	output, err := authService.Authenticate(ctx, domain.AuthenticateInput{
		ProviderType: providerType,
		AuthData:     authData,
	})
	// assertions
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, domain.AccountID(uid), output.AccountID)
	require.True(t, output.IsNew)
}
