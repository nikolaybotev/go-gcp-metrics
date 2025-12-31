package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	gcpmetrics "github.com/nikolaybotev/go-gcp-metrics"
	"github.com/nikolaybotev/go-gcp-metrics/cloud_metadata"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

func main() {
	//  Create info and error loggers
	errorLogger := log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Llongfile)
	infoLogger := log.New(os.Stdout, "INFO: ", log.LstdFlags)

	// Get the project ID from environment variable
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		errorLogger.Fatal("GOOGLE_CLOUD_PROJECT env var must be set")
	}

	// Get the instance ID or hostname to use as a label
	instance := cloud_metadata.GetInstanceName()
	infoLogger.Printf("Using instance: %s", instance)

	// Create the GCP Monitoring client
	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		errorLogger.Fatalf("failed to create metric client: %v", err)
	}
	defer client.Close()

	// Create the MonitoredResource instance
	resource := &monitoredres.MonitoredResource{
		Type: "generic_node",
		Labels: map[string]string{
			"project_id": projectID,
			"location":   "us-central1",
			"namespace":  "",
			"node_id":    instance,
		},
	}

	// Create GcpMetrics and add counters, distributions, and gauges
	metrics := gcpmetrics.NewGcpMetrics(client, projectID, resource, "go/", &gcpmetrics.Options{
		ErrorLogger: errorLogger,
		InfoLogger:  infoLogger,
	})

	// Static metrics (no dynamic label keys - labels fixed at creation time)
	counterA := metrics.Counter("sample_counter_a", map[string]string{"env": "prod"})
	counterB := metrics.Counter("sample_counter_b", map[string]string{"env": "dev"})
	distributionA := metrics.Distribution("sample_distribution_a", "ms", 100, 1, map[string]string{"env": "prod"})
	distributionB := metrics.Distribution("sample_distribution_b", "ms", 50, 5, map[string]string{"env": "dev"})
	gaugeA := metrics.Gauge("sample_gauge_a", map[string]string{"env": "prod"})
	gaugeB := metrics.Gauge("sample_gauge_b", map[string]string{"env": "dev"})

	// Dynamic metrics (with dynamic label keys - label values provided at runtime)
	httpRequests := metrics.Counter("http_requests", nil, "status", "method")
	activeConnections := metrics.Gauge("active_connections", nil, "region")
	requestLatency := metrics.Distribution("request_latency", "ms", 100, 10, nil, "endpoint", "method")

	// Set gauges before emit
	metrics.AddBeforeEmitListener(func() {
		// Set static gauges to random values (no label values needed)
		gaugeA.Set(rand.Int63n(1000))
		gaugeB.Set(rand.Int63n(1000))

		// Set dynamic gauges with different label values
		activeConnections.Set(rand.Int63n(100), "us-east-1")
		activeConnections.Set(rand.Int63n(100), "us-west-2")
		activeConnections.Set(rand.Int63n(100), "eu-west-1")

		infoLogger.Println("Updated gauges")
	})

	// Emit counters every 10 seconds
	ticker := metrics.EmitEvery(ctx, 10*time.Second)
	defer ticker.Stop()

	// Simulate some work and increment counters and gauge
	infoLogger.Println("Starting metrics emission...")
	for {
		// Update static counters (no label values needed)
		counterA.Add(rand.Int63n(100))
		counterB.Add(rand.Int63n(50))

		// Update dynamic counter with various label combinations
		httpRequests.Inc("200", "GET")
		httpRequests.Inc("200", "POST")
		httpRequests.Add(rand.Int63n(10), "404", "GET")
		httpRequests.Add(rand.Int63n(5), "500", "GET")

		infoLogger.Println("Updated counters")

		// Update static distributions with random values (no label values needed)
		for range 100_000 {
			go func() {
				distributionA.Update(rand.Int63n(100))
			}()
			go func() {
				distributionB.Update(rand.Int63n(250))
			}()
		}

		// Update dynamic distribution with different endpoints
		for range 1000 {
			go func() {
				requestLatency.Update(rand.Int63n(500), "/api/users", "GET")
			}()
			go func() {
				requestLatency.Update(rand.Int63n(200), "/api/posts", "GET")
			}()
			go func() {
				requestLatency.Update(rand.Int63n(1000), "/api/users", "POST")
			}()
		}

		time.Sleep(1 * time.Second)
	}
}
