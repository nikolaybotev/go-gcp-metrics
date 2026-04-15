# go-gcp-metrics

A simple go library for publishing metrics to GCP.

## Install

```sh
go get github.com/nikolaybotev/go-gcp-metrics@latest
```

## Usage

See `example/main.go`.

```sh
cd example
go run main.go
```

## Examples

### Static metrics (labels fixed at creation time)

```go
package main

import gcpmetrics "github.com/nikolaybotev/go-gcp-metrics"

func useStatic(metrics gcpmetrics.MetricsCollector) {
	// Static labels are fixed when you create the metric.
	// Any labelValues passed to Inc/Add/Set/Update are ignored for static metrics.
	requests := metrics.Counter("requests_total", map[string]string{"env": "prod"})
	inFlight := metrics.Gauge("in_flight", map[string]string{"env": "prod"})
	latency := metrics.Distribution("request_latency", "ms", 100, 10, map[string]string{"env": "prod"})

	requests.Inc()
	requests.Add(5)

	inFlight.Set(42)

	latency.Update(120)
	latency.Update(250)
}
```

### Dynamic label metrics (label keys fixed; values provided at runtime)

```go
package main

import gcpmetrics "github.com/nikolaybotev/go-gcp-metrics"

func useDynamic(metrics gcpmetrics.MetricsCollector) {
	// Dynamic label KEYS are fixed when you create the metric.
	// Dynamic label VALUES are passed at call sites.
	httpRequests := metrics.Counter("http_requests", nil, "status", "method")
	activeConnections := metrics.Gauge("active_connections", nil, "region")
	requestLatency := metrics.Distribution("request_latency", "ms", 100, 10, nil, "endpoint")

	httpRequests.Inc("200", "GET")
	httpRequests.Add(3, "500", "POST")

	activeConnections.Set(12, "us-east-1")
	activeConnections.Set(7, "eu-west-1")

	requestLatency.Update(123, "/api/users")
	requestLatency.Update(456, "/api/posts")
}
```

### Emit on an interval + set gauges right before emit

```go
package main

import (
	"context"
	"time"

	gcpmetrics "github.com/nikolaybotev/go-gcp-metrics"
)

func run(ctx context.Context, metrics *gcpmetrics.GcpMetrics) {
	inFlight := metrics.Gauge("in_flight", nil)

	metrics.AddBeforeEmitListener(func() {
		// Set gauge values just-in-time before each emit.
		inFlight.Set(123)
	})

	// Starts a goroutine that calls metrics.Emit(ctx) every interval.
	ticker := metrics.EmitEvery(ctx, 10*time.Second)
	defer ticker.Stop()

	<-ctx.Done()
}
```

### Common labels (applied to every emitted metric)

```go
package main

import (
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	gcpmetrics "github.com/nikolaybotev/go-gcp-metrics"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

func newMetrics(client *monitoring.MetricClient, projectID string, mr *monitoredres.MonitoredResource) *gcpmetrics.GcpMetrics {
	return gcpmetrics.NewGcpMetrics(client, projectID, mr, "go/", &gcpmetrics.Options{
		CommonLabels: map[string]string{
			"service": "api",
			"team":    "platform",
		},
	})
}
```

### Local/testing: emit using your own emitter

If you want to collect metrics without a GCP client, you can use `NewMetrics()` and provide your own emitter (e.g. log, Prometheus bridge, tests, etc.).

```go
package main

import (
	"context"
	"log"
	"time"

	gcpmetrics "github.com/nikolaybotev/go-gcp-metrics"
)

type LogEmitter struct{}

func (LogEmitter) Emit(_ context.Context, m *gcpmetrics.Metrics) {
	for _, c := range m.Counters {
		log.Printf("counter %s%v=%d", c.Name, c.Labels, c.Value())
	}
	for _, dc := range m.DynamicCounters {
		for c := range dc.All() {
			log.Printf("counter %s%v=%d", c.Name, c.Labels, c.Value())
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := gcpmetrics.NewMetrics()
	requests := m.Counter("requests_total", nil, "status")
	requests.Inc("200")
	requests.Inc("404")

	ticker := gcpmetrics.ScheduleMetricsEmit(ctx, m, 5*time.Second, LogEmitter{})
	defer ticker.Stop()

	time.Sleep(6 * time.Second)
}
```
