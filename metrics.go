package gcpmetrics

import (
	"context"
	"time"
)

// Metrics defines the public interface for metrics implementations.
type Metrics interface {
	Counter(name string, labels map[string]string) *Counter
	Gauge(name string, labels map[string]string) *Gauge
	Distribution(name, unit string, step, numBuckets int, labels map[string]string) *Distribution
	AddBeforeEmitListener(listener func())
}

type MetricsEmitter interface {
	Emit(ctx context.Context)
	EmitEvery(ctx context.Context, interval time.Duration) *time.Ticker
}
