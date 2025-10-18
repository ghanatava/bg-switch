package controller

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/ghanatava/bg-switch/api/v1alpha1"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// MetricsClient wraps the Prometheus API client
type MetricsClient struct {
	api promv1.API
}

// NewMetricsClient creates and returns a new MetricsClient
func NewMetricsClient(prometheusURL string) (*MetricsClient, error) {
	if prometheusURL == "" {
		// Layer 3: Fallback to common default
		// Assumes Prometheus is in same namespace as operator
		prometheusURL = "http://prometheus:9090"
	}

	client, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	return &MetricsClient{
		api: promv1.NewAPI(client),
	}, nil
}

// QueryMetric executes a PromQL query and returns a single float64 value
func (m *MetricsClient) QueryMetric(ctx context.Context, query string) (float64, error) {
	log := log.FromContext(ctx)

	log.Info("Querying Prometheus", "query", query)

	// Execute the query at current time
	result, warnings, err := m.api.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("error querying prometheus: %w", err)
	}

	// Log any warnings from Prometheus
	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "warnings", warnings)
	}

	// Parse the result based on its type
	switch v := result.(type) {
	case model.Vector:
		// Vector: array of time series with current values
		if len(v) == 0 {
			return 0, fmt.Errorf("no data returned from query")
		}
		// Return the first sample's value
		return float64(v[0].Value), nil

	case *model.Scalar:
		// Scalar: single numeric value
		return float64(v.Value), nil

	default:
		// Unexpected result type
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// AnalyzeHealth checks all configured metrics against their thresholds
// Returns: (isHealthy bool, actualMetrics map, error)
func (m *MetricsClient) AnalyzeHealth(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (bool, map[string]float64, error) {
	log := log.FromContext(ctx)

	// Map to store actual metric values
	metrics := make(map[string]float64)

	// Start optimistic - assume healthy until proven otherwise
	healthy := true

	// Check error rate if configured
	if pd.Spec.Metrics.ErrorRate.Query != "" {
		log.Info("Checking error rate metric")

		errorRate, err := m.QueryMetric(ctx, pd.Spec.Metrics.ErrorRate.Query)
		if err != nil {
			log.Error(err, "Failed to query error rate metric")
			return false, nil, fmt.Errorf("error rate query failed: %w", err)
		}

		// Store the actual value
		metrics["errorRate"] = errorRate

		log.Info("Error rate metric result",
			"value", errorRate,
			"threshold", pd.Spec.Metrics.ErrorRate.Threshold)

		// Check threshold
		if errorRate > pd.Spec.Metrics.ErrorRate.Threshold {
			log.Info("❌ Error rate EXCEEDED threshold - UNHEALTHY",
				"value", errorRate,
				"threshold", pd.Spec.Metrics.ErrorRate.Threshold)
			healthy = false
		} else {
			log.Info("✅ Error rate within threshold - OK")
		}
	}

	// Check latency if configured
	if pd.Spec.Metrics.Latency.Query != "" {
		log.Info("Checking latency metric")

		latency, err := m.QueryMetric(ctx, pd.Spec.Metrics.Latency.Query)
		if err != nil {
			log.Error(err, "Failed to query latency metric")
			return false, nil, fmt.Errorf("latency query failed: %w", err)
		}

		// Store the actual value
		metrics["latency"] = latency

		log.Info("Latency metric result",
			"value", latency,
			"threshold", pd.Spec.Metrics.Latency.Threshold)

		// Check threshold
		if latency > pd.Spec.Metrics.Latency.Threshold {
			log.Info("❌ Latency EXCEEDED threshold - UNHEALTHY",
				"value", latency,
				"threshold", pd.Spec.Metrics.Latency.Threshold)
			healthy = false
		} else {
			log.Info("✅ Latency within threshold - OK")
		}
	}

	// If no metrics configured at all, assume healthy
	if len(metrics) == 0 {
		log.Info("⚠️  No metrics configured - assuming healthy")
		return true, metrics, nil
	}

	// Return final health status
	if healthy {
		log.Info("✅ Overall health: HEALTHY", "metrics", metrics)
	} else {
		log.Info("❌ Overall health: UNHEALTHY", "metrics", metrics)
	}

	return healthy, metrics, nil
}
