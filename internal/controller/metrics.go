package controller

import (
	"fmt"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// MetricsClient wraps the Prometheus API client
type MetricsClient struct {
	api promv1.API
}

// NewMetricsClient creates and returns a new MetricsClient
func NewMetricsClient(prometheusURL string) (*MetricsClient, error) {
	// Use default URL if none provided
	if prometheusURL == "" {
		prometheusURL = "http://prometheus:9090"
	}

	// Create the HTTP client
	client, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	// Wrap it in our struct
	return &MetricsClient{
		api: promv1.NewAPI(client),
	}, nil
}
