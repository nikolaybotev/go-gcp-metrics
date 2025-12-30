package gcpmetrics

import (
	"context"
	"time"
)

type MetricsEmitter interface {
	Emit(ctx context.Context, metrics *Metrics)
}

// ScheduleMetricsEmit schedules the emitter to emit metrics at the given interval in a new goroutine.
// It returns a ticker that can be used to stop the scheduled emissions.
func ScheduleMetricsEmit(
	ctx context.Context,
	metrics *Metrics,
	interval time.Duration,
	emitter MetricsEmitter,
) *time.Ticker {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Notify before emit listeners
				metrics.notifyBeforeEmitListeners()
				// Emit metrics
				emitter.Emit(ctx, metrics)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	return ticker
}
