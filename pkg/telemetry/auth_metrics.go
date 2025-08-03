package telemetry


// AuthMetrics provides authentication-specific metrics
type AuthMetrics struct {
	AuthAttempts    metric.Int64Counter
	AuthSuccess     metric.Int64Counter
	AuthFailures    metric.Int64Counter
	AuthDuration    metric.Float64Histogram
	TokensIssued    metric.Int64Counter
	TokensValidated metric.Int64Counter
}

// NewAuthMetrics creates authentication-specific metrics
func (i *Instrumenter) NewAuthMetrics() (*AuthMetrics, error) {
	authAttempts, err := i.meter.Int64Counter(
		"auth_attempts_total",
		metric.WithDescription("Total number of authentication attempts"),
	)
	if err != nil {
		return nil, err
	}

	authSuccess, err := i.meter.Int64Counter(
		"auth_success_total",
		metric.WithDescription("Total number of successful authentications"),
	)
	if err != nil {
		return nil, err
	}

	authFailures, err := i.meter.Int64Counter(
		"auth_failures_total",
		metric.WithDescription("Total number of failed authentications"),
	)
	if err != nil {
		return nil, err
	}

	authDuration, err := i.meter.Float64Histogram(
		"auth_duration_seconds",
		metric.WithDescription("Duration of authentication operations in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	tokensIssued, err := i.meter.Int64Counter(
		"tokens_issued_total",
		metric.WithDescription("Total number of tokens issued"),
	)
	if err != nil {
		return nil, err
	}

	tokensValidated, err := i.meter.Int64Counter(
		"tokens_validated_total",
		metric.WithDescription("Total number of tokens validated"),
	)
	if err != nil {
		return nil, err
	}

	return &AuthMetrics{
		AuthAttempts:    authAttempts,
		AuthSuccess:     authSuccess,
		AuthFailures:    authFailures,
		AuthDuration:    authDuration,
		TokensIssued:    tokensIssued,
		TokensValidated: tokensValidated,
	}, nil
}
