package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
)

func main() {
	// Get the project ID from environment variable
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT env var must be set")
	}

	// Get the instance ID or hostname to use as a label
	instance, err := GetInstanceName()
	if err != nil {
		log.Fatalf("failed to get instance: %v", err)
	}
	log.Printf("Using instance: %s", instance)
	commonLabels := map[string]string{
		"instance": instance,
	}

	// Create the GCP Monitoring client
	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("failed to create metric client: %v", err)
	}
	defer client.Close()

	// Create MetricsEmitter and add counters and distributions
	emitter := NewMetricsEmitter(client, projectID, "go/", commonLabels)
	counterA := emitter.Counter("sample_counter_a", map[string]string{"env": "prod"})
	counterB := emitter.Counter("sample_counter_b", map[string]string{"env": "dev"})
	distributionA := emitter.Distribution("sample_distribution_a", "ms", 100, 1, map[string]string{"env": "prod"})
	distributionB := emitter.Distribution("sample_distribution_b", "ms", 50, 5, map[string]string{"env": "dev"})

	// Emit counters every 10 seconds
	ticker := emitter.EmitEvery(10 * time.Second)
	defer ticker.Stop()

	// Simulate some work and increment counters
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
