package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT env var must be set")
	}

	// Create two sample counters
	counterA := NewCounterWithLabels("sample_counter_a", map[string]string{"env": "prod"})
	counterB := NewCounterWithLabels("sample_counter_b", map[string]string{"env": "dev"})

	// Create MetricsEmitter and add counters
	emitter := NewMetricsEmitter(projectID)
	emitter.AddCounter(counterA)
	emitter.AddCounter(counterB)

	for {
		// Simulate incrementing counters
		counterA.Add(rand.Int63n(100))
		counterB.Add(rand.Int63n(50))

		// Emit all counters to GCP Cloud Monitoring
		emitter.Emit()

		fmt.Printf("Emitted counters: %s=%d, %s=%d\n",
			counterA.Name, counterA.Value(),
			counterB.Name, counterB.Value(),
		)

		time.Sleep(10 * time.Second)
	}
}
