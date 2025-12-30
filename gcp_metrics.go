package gcpmetrics

import (
	"context"
	"log"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

// Options contains optional configuration for GcpMetrics.
type Options struct {
	ErrorLogger  *log.Logger
	InfoLogger   *log.Logger
	CommonLabels map[string]string
}

// GcpMetrics is a Metrics implementation that emits metrics to Google Cloud Monitoring.
// It composes Metrics for metric collection and GcpMetricsEmitter for emission.
type GcpMetrics struct {
	*Metrics
	*GcpMetricsEmitter
}

// NewGcpMetrics creates a new GcpMetrics instance.
func NewGcpMetrics(
	client *monitoring.MetricClient,
	projectID string,
	monitoredResource *monitoredres.MonitoredResource,
	metricsNamePrefix string,
	opts *Options,
) *GcpMetrics {
	return &GcpMetrics{
		Metrics:           NewMetrics(),
		GcpMetricsEmitter: NewGcpMetricsEmitter(client, projectID, monitoredResource, metricsNamePrefix, opts),
	}
}

// Emit delegates to the emitter's Emit method, passing the embedded Metrics.
func (me *GcpMetrics) Emit(ctx context.Context) {
	me.GcpMetricsEmitter.Emit(ctx, me.Metrics)
}

// EmitEvery delegates to the emitter's EmitEvery method, passing the embedded Metrics.
func (me *GcpMetrics) EmitEvery(ctx context.Context, interval time.Duration) *time.Ticker {
	return ScheduleMetricsEmit(ctx, me.Metrics, interval, me.GcpMetricsEmitter)
}
