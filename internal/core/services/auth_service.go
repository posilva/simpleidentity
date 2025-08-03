package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/posilva/simpleidentity/internal/core/domain"
	"github.com/posilva/simpleidentity/internal/core/ports"
	"github.com/posilva/simpleidentity/pkg/telemetry"
)

// AuthService is the implementation of the AuthService interface.
type authService struct {
	providerFactory ports.AuthProviderFactory
	repository      ports.AccountsRepository
	tracer          trace.Tracer
	authMetrics     *telemetry.AuthMetrics
}

// Safegard check to ensure authService implements the AuthService interface
var _ ports.AuthService = (*authService)(nil)

// NewAuthService creates a new instance of AuthService with the given provider factory.
func NewAuthService(providerFactory ports.AuthProviderFactory, r ports.AccountsRepository) *authService {
	instrumenter := telemetry.NewInstrumenter("auth_service")
	authMetrics, _ := instrumenter.NewAuthMetrics() // Ignore error for now, metrics are optional
	
	return &authService{
		providerFactory: providerFactory,
		repository:      r,
		tracer:          otel.Tracer("auth_service"),
		authMetrics:     authMetrics,
	}
}

// Authenticate authenticates a user using the specified authentication provider.
func (s *authService) Authenticate(ctx context.Context, input domain.AuthenticateInput) (*domain.AuthenticateOutput, error) {
	start := time.Now()
	
	// Start tracing span
	ctx, span := s.tracer.Start(ctx, "auth_service.authenticate",
		trace.WithAttributes(
			telemetry.ProviderAttr(string(input.ProviderType)),
			telemetry.OperationAttr("authenticate"),
		),
	)
	defer span.End()

	// Record authentication attempt
	if s.authMetrics != nil {
		s.authMetrics.AuthAttempts.Add(ctx, 1, 
			telemetry.ProviderAttr(string(input.ProviderType)),
		)
	}

	provider, err := s.providerFactory.Get(input.ProviderType)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get provider")
		span.RecordError(err, trace.WithAttributes(
			attribute.String("error.phase", "provider_factory"),
		))
		
		if s.authMetrics != nil {
			s.authMetrics.AuthFailures.Add(ctx, 1,
				telemetry.ProviderAttr(string(input.ProviderType)),
				attribute.String("failure_reason", "provider_not_found"),
			)
		}
		return nil, err
	}

	result, err := provider.Authenticate(ctx, input.AuthData)
	if err != nil {
		span.SetStatus(codes.Error, "provider authentication failed")
		span.RecordError(err, trace.WithAttributes(
			attribute.String("error.phase", "provider_authenticate"),
		))
		
		if s.authMetrics != nil {
			s.authMetrics.AuthFailures.Add(ctx, 1,
				telemetry.ProviderAttr(string(input.ProviderType)),
				attribute.String("failure_reason", "provider_auth_failed"),
			)
		}
		return nil, err
	}

	// Add provider-specific attributes
	span.SetAttributes(
		attribute.String("provider.user_id", result.GetID()),
	)

	accountID, err := s.repository.ResolveIDByProvider(ctx, input.ProviderType, result.GetID())
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			// this means that the account does not exist, so we need to create it
			span.AddEvent("creating_new_account", trace.WithAttributes(
				attribute.String("provider.user_id", result.GetID()),
			))
			
			accountID, err := s.repository.Create(ctx, input.ProviderType, result.GetID())
			if err != nil {
				span.SetStatus(codes.Error, "failed to create account")
				span.RecordError(err, trace.WithAttributes(
					attribute.String("error.phase", "account_creation"),
				))
				
				if s.authMetrics != nil {
					s.authMetrics.AuthFailures.Add(ctx, 1,
						telemetry.ProviderAttr(string(input.ProviderType)),
						attribute.String("failure_reason", "account_creation_failed"),
					)
				}
				return nil, fmt.Errorf("failed to create account: %w", err)
			}
			
			// Record successful authentication with new account
			span.SetAttributes(
				telemetry.UserIDAttr(string(accountID)),
				attribute.Bool("account.is_new", true),
			)
			span.SetStatus(codes.Ok, "authentication successful with new account")
			
			if s.authMetrics != nil {
				duration := time.Since(start).Seconds()
				s.authMetrics.AuthSuccess.Add(ctx, 1,
					telemetry.ProviderAttr(string(input.ProviderType)),
					attribute.Bool("account.is_new", true),
				)
				s.authMetrics.AuthDuration.Record(ctx, duration,
					telemetry.ProviderAttr(string(input.ProviderType)),
					telemetry.StatusAttr("success"),
				)
			}
			
			return &domain.AuthenticateOutput{
				AccountID: accountID,
				IsNew:     true,
			}, nil
		}
		
		span.SetStatus(codes.Error, "failed to resolve account ID")
		span.RecordError(err, trace.WithAttributes(
			attribute.String("error.phase", "account_resolution"),
		))
		
		if s.authMetrics != nil {
			s.authMetrics.AuthFailures.Add(ctx, 1,
				telemetry.ProviderAttr(string(input.ProviderType)),
				attribute.String("failure_reason", "account_resolution_failed"),
			)
		}
		return nil, fmt.Errorf("failed to resolve account ID: %w", err)
	}

	// Record successful authentication with existing account
	span.SetAttributes(
		telemetry.UserIDAttr(string(accountID)),
		attribute.Bool("account.is_new", false),
	)
	span.SetStatus(codes.Ok, "authentication successful")
	
	if s.authMetrics != nil {
		duration := time.Since(start).Seconds()
		s.authMetrics.AuthSuccess.Add(ctx, 1,
			telemetry.ProviderAttr(string(input.ProviderType)),
			attribute.Bool("account.is_new", false),
		)
		s.authMetrics.AuthDuration.Record(ctx, duration,
			telemetry.ProviderAttr(string(input.ProviderType)),
			telemetry.StatusAttr("success"),
		)
	}

	return &domain.AuthenticateOutput{
		AccountID: accountID,
	}, nil
}
