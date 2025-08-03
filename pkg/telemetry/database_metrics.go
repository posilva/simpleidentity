package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// DatabaseMetrics provides database-specific metrics
type DatabaseMetrics struct {
	ConnectionsActive metric.Int64UpDownCounter
	ConnectionsOpened metric.Int64Counter
	QueryDuration     metric.Float64Histogram
	QueryCount        metric.Int64Counter
	QueryErrors       metric.Int64Counter
}

// NewDatabaseMetrics creates database-specific metrics
func (i *Instrumenter) NewDatabaseMetrics(dbName string) (*DatabaseMetrics, error) {
	connectionsActive, err := i.meter.Int64UpDownCounter(
		fmt.Sprintf("db_%s_connections_active", dbName),
		metric.WithDescription("Number of active database connections"),
	)
	if err != nil {
		return nil, err
	}

	connectionsOpened, err := i.meter.Int64Counter(
		fmt.Sprintf("db_%s_connections_opened_total", dbName),
		metric.WithDescription("Total number of database connections opened"),
	)
	if err != nil {
		return nil, err
	}

	queryDuration, err := i.meter.Float64Histogram(
		fmt.Sprintf("db_%s_query_duration_seconds", dbName),
		metric.WithDescription("Duration of database queries in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	queryCount, err := i.meter.Int64Counter(
		fmt.Sprintf("db_%s_queries_total", dbName),
		metric.WithDescription("Total number of database queries"),
	)
	if err != nil {
		return nil, err
	}

	queryErrors, err := i.meter.Int64Counter(
		fmt.Sprintf("db_%s_query_errors_total", dbName),
		metric.WithDescription("Total number of database query errors"),
	)
	if err != nil {
		return nil, err
	}

	return &DatabaseMetrics{
		ConnectionsActive: connectionsActive,
		ConnectionsOpened: connectionsOpened,
		QueryDuration:     queryDuration,
		QueryCount:        queryCount,
		QueryErrors:       queryErrors,
	}, nil
}