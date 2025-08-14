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
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT env var must be set")
	}

	hostname, err := GetInstanceIdOrHostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	log.Printf("Using hostname: %s", hostname)

	commonLabels := map[string]string{
		"hostname": hostname,
	}

	// Create the GCP Monitoring client
	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("failed to create metric client: %v", err)
	}
	defer client.Close()

	// Create MetricsEmitter and add counters
	emitter := NewMetricsEmitter(client, projectID, "", commonLabels)
	counterA := emitter.Counter("sample_counter_a", map[string]string{"env": "prod"})
	counterB := emitter.Counter("sample_counter_b", map[string]string{"env": "dev"})

	// Emit counters every 10 seconds
	ticker := emitter.EmitEvery(10 * time.Second)
	defer ticker.Stop()

	log.Println("Starting metrics emission...")

	for {
		// Simulate incrementing counters
		counterA.Add(rand.Int63n(100))
		counterB.Add(rand.Int63n(50))

		fmt.Printf("Updated counters: %s=%d, %s=%d\n",
			counterA.Name, counterA.Value(),
			counterB.Name, counterB.Value(),
		)

		time.Sleep(1 * time.Second)
	}
}
