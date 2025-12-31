package gcpmetrics

import (
	"sync"
	"testing"
)

func TestDynamicCounter_Basic(t *testing.T) {
	counter := NewDynamicCounter("test_counter", "status", "method")

	// Increment with different label combinations
	counter.Inc("200", "GET")
	counter.Inc("200", "GET")
	counter.Inc("404", "GET")
	counter.Add(5, "200", "POST")

	// Verify values
	if v := counter.Value("200", "GET"); v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
	if v := counter.Value("404", "GET"); v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
	if v := counter.Value("200", "POST"); v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
}

func TestDynamicCounter_All(t *testing.T) {
	counter := NewDynamicCounter("test_counter", "status")

	counter.Inc("200")
	counter.Inc("404")
	counter.Inc("500")

	count := 0
	for c := range counter.All() {
		if c.Value() != 1 {
			t.Errorf("expected value 1, got %d", c.Value())
		}
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 counters, got %d", count)
	}
}

func TestDynamicGauge_Basic(t *testing.T) {
	gauge := NewDynamicGauge("test_gauge", "location", "sensor")

	gauge.Set(25, "datacenter1", "sensor1")
	gauge.Set(30, "datacenter1", "sensor2")
	gauge.Set(20, "datacenter2", "sensor1")

	if v := gauge.Value("datacenter1", "sensor1"); v != 25 {
		t.Errorf("expected 25, got %d", v)
	}
	if v := gauge.Value("datacenter1", "sensor2"); v != 30 {
		t.Errorf("expected 30, got %d", v)
	}
	if v := gauge.Value("datacenter2", "sensor1"); v != 20 {
		t.Errorf("expected 20, got %d", v)
	}
}

func TestDynamicGauge_All(t *testing.T) {
	gauge := NewDynamicGauge("test_gauge", "location")

	gauge.Set(10, "loc1")
	gauge.Set(20, "loc2")

	count := 0
	for range gauge.All() {
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 gauges, got %d", count)
	}
}

func TestDynamicDistribution_Basic(t *testing.T) {
	dist := NewDynamicDistribution("test_dist", "ms", 100, 10, "endpoint")

	dist.Update(50, "/api/users")
	dist.Update(150, "/api/users")
	dist.Update(250, "/api/posts")

	count := 0
	for range dist.All() {
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 distributions, got %d", count)
	}
}

func TestDynamicCounter_Concurrent(t *testing.T) {
	counter := NewDynamicCounter("concurrent_counter", "goroutine")

	var wg sync.WaitGroup
	numGoroutines := 100
	incrementsPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			label := "same_label"
			for j := 0; j < incrementsPerGoroutine; j++ {
				counter.Inc(label)
			}
		}(i)
	}

	wg.Wait()

	expected := int64(numGoroutines * incrementsPerGoroutine)
	if v := counter.Value("same_label"); v != expected {
		t.Errorf("expected %d, got %d", expected, v)
	}
}

func TestDynamicCounter_ConcurrentDifferentLabels(t *testing.T) {
	counter := NewDynamicCounter("concurrent_counter", "id")

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			label := string(rune('A' + (id % 26)))
			for j := 0; j < 100; j++ {
				counter.Inc(label)
			}
		}(i)
	}

	wg.Wait()

	// Verify all counters exist and have correct values
	totalCount := 0
	for c := range counter.All() {
		if c.Value() <= 0 {
			t.Errorf("counter should have positive value, got %d", c.Value())
		}
		totalCount++
	}

	// We have 26 unique labels (A-Z)
	if totalCount != 26 {
		t.Errorf("expected 26 unique counters, got %d", totalCount)
	}
}

func TestLabelValuesToMap(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		values   []string
		expected map[string]string
	}{
		{
			name:     "matching lengths",
			keys:     []string{"status", "method"},
			values:   []string{"200", "GET"},
			expected: map[string]string{"status": "200", "method": "GET"},
		},
		{
			name:     "fewer values than keys",
			keys:     []string{"status", "method"},
			values:   []string{"200"},
			expected: map[string]string{"status": "200"},
		},
		{
			name:     "more values than keys",
			keys:     []string{"status"},
			values:   []string{"200", "GET"},
			expected: map[string]string{"status": "200"},
		},
		{
			name:     "empty values",
			keys:     []string{"status", "method"},
			values:   []string{},
			expected: map[string]string{},
		},
		{
			name:     "empty keys",
			keys:     []string{},
			values:   []string{"200", "GET"},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := labelValuesToMap(tt.keys, tt.values)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestMetrics_WithDynamicLabels(t *testing.T) {
	metrics := NewMetrics()

	// Create dynamic metrics
	counter := metrics.CounterWithLabels("requests", "status", "method")
	gauge := metrics.GaugeWithLabels("temperature", "location")
	dist := metrics.DistributionWithLabels("latency", "ms", 100, 10, "endpoint")

	// Use them
	counter.Inc("200", "GET")
	counter.Inc("404", "GET")
	gauge.Set(25, "datacenter1")
	dist.Update(50, "/api/users")

	// Verify they're stored in metrics
	if len(metrics.DynamicCounters) != 1 {
		t.Errorf("expected 1 dynamic counter, got %d", len(metrics.DynamicCounters))
	}
	if len(metrics.DynamicGauges) != 1 {
		t.Errorf("expected 1 dynamic gauge, got %d", len(metrics.DynamicGauges))
	}
	if len(metrics.DynamicDistributions) != 1 {
		t.Errorf("expected 1 dynamic distribution, got %d", len(metrics.DynamicDistributions))
	}

	// Verify values
	if v := counter.Value("200", "GET"); v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
	if v := gauge.Value("datacenter1"); v != 25 {
		t.Errorf("expected 25, got %d", v)
	}
}
