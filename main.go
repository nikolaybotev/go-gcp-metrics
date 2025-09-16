package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

func main() {
	// Get the project ID from environment variable
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT env var must be set")
	}

	// Get the instance ID or hostname to use as a label
	instance := GetInstanceName()
	log.Printf("Using instance: %s", instance)
	commonLabels := map[string]string{}

	// Create the GCP Monitoring client
	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("failed to create metric client: %v", err)
	}
	defer client.Close()

	// Create the MonitoredResource instance
	resource := &monitoredres.MonitoredResource{
		Type: "generic_node",
		Labels: map[string]string{
			"project_id": projectID,
			"location":   "us-central1",
			"namespace":  "",
			"node_name":  instance,
		},
	}

	// Create GcpMetrics and add counters, distributions, and gauges
	metrics := NewGcpMetrics(client, projectID, resource, "go/", commonLabels)
	counterA := metrics.Counter("sample_counter_a", map[string]string{"env": "prod"})
	counterB := metrics.Counter("sample_counter_b", map[string]string{"env": "dev"})
	distributionA := metrics.Distribution("sample_distribution_a", "ms", 100, 1, map[string]string{"env": "prod"})
	distributionB := metrics.Distribution("sample_distribution_b", "ms", 50, 5, map[string]string{"env": "dev"})
	gaugeA := metrics.Gauge("sample_gauge_a", map[string]string{"env": "prod"})
	gaugeB := metrics.Gauge("sample_gauge_b", map[string]string{"env": "dev"})

	// Set gauges before emit
	metrics.AddBeforeEmitListener(func() {
		// Set gauge to a random value
		gaugeA.Set(rand.Int63n(1000))
		gaugeB.Set(rand.Int63n(1000))

		fmt.Printf("Updated gauges: %s=%d, %s=%d\n",
			gaugeA.Name, gaugeA.Value(),
			gaugeB.Name, gaugeB.Value(),
		)
	})

	// Emit counters every 10 seconds
	ticker := metrics.EmitEvery(ctx, 10*time.Second)
	defer ticker.Stop()

	// Simulate some work and increment counters and gauge
	log.Println("Starting metrics emission...")
	for {
		counterA.Add(rand.Int63n(100))
		counterB.Add(rand.Int63n(50))

		fmt.Printf("Updated counters: %s=%d, %s=%d\n",
			counterA.Name, counterA.Value(),
			counterB.Name, counterB.Value(),
		)

		// Update distributions with random values
		for i := 0; i < 100_000; i++ {
			go func() {
				distributionA.Update(rand.Int63n(100))
			}()
			go func() {
				distributionB.Update(rand.Int63n(250))
			}()
		}

		time.Sleep(1 * time.Second)
	}
}
