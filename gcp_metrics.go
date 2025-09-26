package gcpmetrics

import (
	"context"
	"io"
	"log"
	"math"
	"path"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/genproto/googleapis/api/distribution"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GcpMetrics struct {
	Client              *monitoring.MetricClient
	ProjectID           string
	MonitoredResource   *monitoredres.MonitoredResource
	MetricsNamePrefix   string
	CommonLabels        []Label
	Counters            []*Counter
	Distributions       []*Distribution
	Gauges              []*Gauge
	BeforeEmitListeners []func()
	errorLogger         *log.Logger
	infoLogger          *log.Logger
}

func NewGcpMetrics(
	client *monitoring.MetricClient,
	projectID string,
	monitoredResource *monitoredres.MonitoredResource,
	metricsNamePrefix string,
	errorLogger *log.Logger,
	infoLogger *log.Logger,
	commonLabels ...Label,
) *GcpMetrics {
	// Set defaults if nil
	if errorLogger == nil {
		errorLogger = log.Default()
	}
	if infoLogger == nil {
		infoLogger = log.New(io.Discard, "", 0)
	}

	return &GcpMetrics{
		Client:              client,
		ProjectID:           projectID,
		MonitoredResource:   monitoredResource,
		MetricsNamePrefix:   metricsNamePrefix,
		CommonLabels:        commonLabels,
		Counters:            []*Counter{},
		Distributions:       []*Distribution{},
		Gauges:              []*Gauge{},
		BeforeEmitListeners: []func(){},
		errorLogger:         errorLogger,
		infoLogger:          infoLogger,
	}
}

// AddCounter addds a Counter to the metrics.
func (me *GcpMetrics) AddCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

// Counter creates a new Counter, adds it to the metrics, and returns it.
func (me *GcpMetrics) Counter(name string, labels ...Label) *Counter {
	counter := NewCounter(name, labels...)
	me.AddCounter(counter)
	return counter
}

// AddGauge adds a Gauge to the metrics.
func (me *GcpMetrics) AddGauge(g *Gauge) {
	me.Gauges = append(me.Gauges, g)
}

// Gauge creates a new Gauge, adds it to the metrics, and returns it.
func (me *GcpMetrics) Gauge(name string, labels ...Label) *Gauge {
	g := NewGauge(name, labels...)
	me.AddGauge(g)
	return g
}

// AddDistribution adds a Distribution to the metrics.
func (me *GcpMetrics) AddDistribution(dist *Distribution) {
	me.Distributions = append(me.Distributions, dist)
}

// Distribution creates a new Distribution, adds it to the metrics, and returns it.
func (me *GcpMetrics) Distribution(
	name,
	unit string,
	step,
	numBuckets int,
	labels ...Label,
) *Distribution {
	dist := NewDistribution(name, unit, step, numBuckets, labels...)
	me.AddDistribution(dist)
	return dist
}

func (me *GcpMetrics) AddBeforeEmitListener(listener func()) {
	me.BeforeEmitListeners = append(me.BeforeEmitListeners, listener)
}

// mergeLabels merges common labels with metric-specific labels.
func (me *GcpMetrics) mergeLabels(specificLabels []Label) map[string]string {
	labels := make(map[string]string, len(me.CommonLabels)+len(specificLabels))
	for _, v := range me.CommonLabels {
		labels[v.Name] = v.Value
	}
	for _, v := range specificLabels {
		labels[v.Name] = v.Value
	}
	return labels
}

// buildMetric constructs a metric.Metric with the correct type and merged labels.
func (me *GcpMetrics) buildMetric(name string, specificLabels []Label) *metric.Metric {
	return &metric.Metric{
		Type:   "custom.googleapis.com/" + path.Join(me.MetricsNamePrefix, name),
		Labels: me.mergeLabels(specificLabels),
	}
}

func (me *GcpMetrics) Emit(ctx context.Context) {
	if me.Client == nil {
		me.errorLogger.Println("Client must be set in GcpMetrics")
		return
	}
	if me.ProjectID == "" {
		me.errorLogger.Println("ProjectID must be set in GcpMetrics")
		return
	}
	if me.MonitoredResource == nil {
		me.errorLogger.Println("MonitoredResource must be set in GcpMetrics")
		return
	}

	now := time.Now()
	interval := &monitoringpb.TimeInterval{
		EndTime: timestamppb.New(now),
	}

	var timeSeriesList []*monitoringpb.TimeSeries

	// Emit counters
	for _, counter := range me.Counters {
		value := counter.Value()

		ts := &monitoringpb.TimeSeries{
			Metric:   me.buildMetric(counter.Name, counter.Labels),
			Resource: me.MonitoredResource,
			Points: []*monitoringpb.Point{
				{
					Interval: interval,
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_Int64Value{
							Int64Value: value,
						},
					},
				},
			},
		}

		timeSeriesList = append(timeSeriesList, ts)
	}

	// Emit gauges
	for _, gauge := range me.Gauges {
		value := gauge.Value()

		ts := &monitoringpb.TimeSeries{
			Metric:   me.buildMetric(gauge.Name, gauge.Labels),
			Resource: me.MonitoredResource,
			Points: []*monitoringpb.Point{
				{
					Interval: interval,
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_Int64Value{
							Int64Value: value,
						},
					},
				},
			},
		}

		timeSeriesList = append(timeSeriesList, ts)
	}

	// Emit distributions
	for _, dist := range me.Distributions {
		value := dist.GetAndClear()
		if value.NumSamples == 0 {
			continue
		}

		ts := &monitoringpb.TimeSeries{
			Metric:   me.buildMetric(dist.Name, dist.Labels),
			Unit:     dist.Unit,
			Resource: me.MonitoredResource,
			Points: []*monitoringpb.Point{
				{
					Interval: interval,
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_DistributionValue{
							DistributionValue: &distribution.Distribution{
								Count:                 value.NumSamples,
								Mean:                  value.Mean,
								SumOfSquaredDeviation: value.SumOfSquaredDeviation,
								BucketOptions: &distribution.Distribution_BucketOptions{
									Options: &distribution.Distribution_BucketOptions_ExplicitBuckets{
										ExplicitBuckets: &distribution.Distribution_BucketOptions_Explicit{
											Bounds: dist.BucketBounds(),
										},
									},
								},
								BucketCounts: value.Buckets,
							},
						},
					},
				},
			},
		}

		timeSeriesList = append(timeSeriesList, ts)
	}

	if len(timeSeriesList) == 0 {
		return
	}

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + me.ProjectID,
		TimeSeries: timeSeriesList,
	}

	if err := me.Client.CreateTimeSeries(ctx, req); err != nil {
		me.errorLogger.Printf("failed to write time series data: %v", err)
	} else {
		for _, ts := range timeSeriesList {
			metricName := ts.Metric.Type
			// Remove the "custom.googleapis.com/" prefix
			if len(metricName) > 22 {
				metricName = metricName[22:]
			}
			// Remove the MetricsNamePrefix
			if me.MetricsNamePrefix != "" && len(metricName) > len(me.MetricsNamePrefix)+1 {
				if len(metricName) > len(me.MetricsNamePrefix) && metricName[:len(me.MetricsNamePrefix)] == me.MetricsNamePrefix {
					metricName = metricName[len(me.MetricsNamePrefix):]
				}
			}

			// Add labels in square brackets
			if len(ts.Metric.Labels) > 0 {
				labelParts := make([]string, 0, len(ts.Metric.Labels))
				for k, v := range ts.Metric.Labels {
					labelParts = append(labelParts, k+"="+v)
				}
				metricName += "[" + strings.Join(labelParts, ",") + "]"
			}

			if len(ts.Points) > 0 {
				point := ts.Points[0]
				switch v := point.Value.Value.(type) {
				case *monitoringpb.TypedValue_Int64Value:
					me.infoLogger.Printf("Published metric %s value %d", metricName, v.Int64Value)
				case *monitoringpb.TypedValue_DistributionValue:
					dist := v.DistributionValue
					// Calculate standard deviation from sum of squared deviations
					var stdDev float64
					if dist.Count > 1 {
						variance := dist.SumOfSquaredDeviation / float64(dist.Count-1)
						stdDev = math.Sqrt(variance)
					}
					me.infoLogger.Printf("Published distribution %s with %d samples (mean %.2f, stddev %.2f)",
						metricName, dist.Count, dist.Mean, stdDev)
				}
			}
		}
	}
}

// EmitEvery schedules Emit to run at the given interval in a new goroutine.
func (me *GcpMetrics) EmitEvery(ctx context.Context, interval time.Duration) *time.Ticker {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Notify before emit listeners
				for _, listener := range me.BeforeEmitListeners {
					if listener != nil {
						listener()
					}
				}
				// Emit metrics
				me.Emit(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	return ticker
}
