package main

import (
	"context"
	"fmt"
	"log"
	"path"
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
	CommonLabels        map[string]string
	Counters            []*Counter
	Distributions       []*Distribution
	Gauges              []*Gauge
	BeforeEmitListeners []func()
}

func NewGcpMetrics(
	client *monitoring.MetricClient,
	projectID string,
	monitoredResource *monitoredres.MonitoredResource,
	metricsNamePrefix string,
	commonLabels map[string]string,
) *GcpMetrics {
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
	}
}

// AddCounter addds a Counter to the metrics.
func (me *GcpMetrics) AddCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

// Counter creates a new Counter, adds it to the metrics, and returns it.
func (me *GcpMetrics) Counter(name string, labels map[string]string) *Counter {
	counter := NewCounter(name, labels)
	me.AddCounter(counter)
	return counter
}

// AddGauge adds a Gauge to the metrics.
func (me *GcpMetrics) AddGauge(g *Gauge) {
	me.Gauges = append(me.Gauges, g)
}

// Gauge creates a new Gauge, adds it to the metrics, and returns it.
func (me *GcpMetrics) Gauge(name string, labels map[string]string) *Gauge {
	g := NewGauge(name, labels)
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
	labels map[string]string,
) *Distribution {
	dist := NewDistribution(name, unit, step, numBuckets, labels)
	me.AddDistribution(dist)
	return dist
}

func (me *GcpMetrics) AddBeforeEmitListener(listener func()) {
	me.BeforeEmitListeners = append(me.BeforeEmitListeners, listener)
}

// mergeLabels merges common labels with metric-specific labels.
func (me *GcpMetrics) mergeLabels(specific map[string]string) map[string]string {
	labels := make(map[string]string, len(me.CommonLabels)+len(specific))
	for k, v := range me.CommonLabels {
		labels[k] = v
	}
	for k, v := range specific {
		labels[k] = v
	}
	return labels
}

// buildMetric constructs a metric.Metric with the correct type and merged labels.
func (me *GcpMetrics) buildMetric(name string, specificLabels map[string]string) *metric.Metric {
	return &metric.Metric{
		Type:   "custom.googleapis.com/" + path.Join(me.MetricsNamePrefix, name),
		Labels: me.mergeLabels(specificLabels),
	}
}

func (me *GcpMetrics) Emit(ctx context.Context) {
	if me.Client == nil {
		log.Println("Client must be set in GcpMetrics")
		return
	}
	if me.ProjectID == "" {
		log.Println("ProjectID must be set in GcpMetrics")
		return
	}
	if me.MonitoredResource == nil {
		log.Println("MonitoredResource must be set in GcpMetrics")
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

		ts := &monitoringpb.TimeSeries{
			Metric:   me.buildMetric(dist.Name, dist.Labels),
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
		log.Printf("failed to write time series data: %v", err)
	} else {
		for _, counter := range me.Counters {
			fmt.Printf("Published counter %s value %d at %s\n", counter.Name, counter.Value(), now.Format(time.RFC3339))
		}
		for _, gauge := range me.Gauges {
			fmt.Printf("Published gauge %s value %d at %s\n", gauge.Name, gauge.Value(), now.Format(time.RFC3339))
		}
		for _, dist := range me.Distributions {
			fmt.Printf("Published distribution %s at %s\n", dist.Name, now.Format(time.RFC3339))
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
