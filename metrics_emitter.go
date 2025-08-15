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

type MetricsEmitter struct {
	Client              *monitoring.MetricClient
	ProjectID           string
	MetricsNamePrefix   string
	CommonLabels        map[string]string
	Counters            []*Counter
	Distributions       []*Distribution
	Gauges              []*Gauge
	BeforeEmitListeners []func()
}

func NewMetricsEmitter(
	client *monitoring.MetricClient,
	projectID string,
	metricsNamePrefix string,
	commonLabels map[string]string,
) *MetricsEmitter {
	return &MetricsEmitter{
		Client:              client,
		ProjectID:           projectID,
		MetricsNamePrefix:   metricsNamePrefix,
		CommonLabels:        commonLabels,
		Counters:            []*Counter{},
		Distributions:       []*Distribution{},
		Gauges:              []*Gauge{},
		BeforeEmitListeners: []func(){},
	}
}

// AddCounter addds a Counter to the emitter.
func (me *MetricsEmitter) AddCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

// Counter creates a new Counter, adds it to the emitter, and returns it.
func (me *MetricsEmitter) Counter(name string, labels map[string]string) *Counter {
	counter := NewCounterWithLabels(name, labels)
	me.AddCounter(counter)
	return counter
}

// AddDistribution adds a Distribution to the emitter.
func (me *MetricsEmitter) AddDistribution(dist *Distribution) {
	me.Distributions = append(me.Distributions, dist)
}

// Distribution creates a new Distribution, adds it to the emitter, and returns it.
func (me *MetricsEmitter) Distribution(
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

// AddGauge adds a Gauge to the emitter.
func (me *MetricsEmitter) AddGauge(g *Gauge) {
	me.Gauges = append(me.Gauges, g)
}

// Gauge creates a new Gauge, adds it to the emitter, and returns it.
func (me *MetricsEmitter) Gauge(name string, labels map[string]string) *Gauge {
	g := NewGauge(name, labels)
	me.AddGauge(g)
	return g
}

func (me *MetricsEmitter) AddBeforeEmitListener(listener func()) {
	me.BeforeEmitListeners = append(me.BeforeEmitListeners, listener)
}

func (me *MetricsEmitter) Emit() {
	if me.ProjectID == "" {
		log.Println("ProjectID must be set in MetricsEmitter")
		return
	}
	if me.Client == nil {
		log.Println("Metric client is not initialized")
		return
	}

	ctx := context.Background()

	resourceType := "global"
	now := time.Now()

	var timeSeriesList []*monitoringpb.TimeSeries

	// Emit counters
	for _, counter := range me.Counters {
		value := counter.Value()

		// Merge common labels and counter labels
		labels := make(map[string]string)
		for k, v := range me.CommonLabels {
			labels[k] = v
		}
		for k, v := range counter.Labels {
			labels[k] = v
		}

		metricType := "custom.googleapis.com/" + path.Join(me.MetricsNamePrefix, counter.Name)

		ts := &monitoringpb.TimeSeries{
			Metric: &metric.Metric{
				Type:   metricType,
				Labels: labels,
			},
			Resource: &monitoredres.MonitoredResource{
				Type: resourceType,
				Labels: map[string]string{
					"project_id": me.ProjectID,
				},
			},
			Points: []*monitoringpb.Point{
				{
					Interval: &monitoringpb.TimeInterval{
						EndTime: timestamppb.New(now),
					},
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

		// Merge common labels and gauge labels
		labels := make(map[string]string)
		for k, v := range me.CommonLabels {
			labels[k] = v
		}
		for k, v := range gauge.Labels {
			labels[k] = v
		}

		metricType := "custom.googleapis.com/" + path.Join(me.MetricsNamePrefix, gauge.Name)

		ts := &monitoringpb.TimeSeries{
			Metric: &metric.Metric{
				Type:   metricType,
				Labels: labels,
			},
			Resource: &monitoredres.MonitoredResource{
				Type: resourceType,
				Labels: map[string]string{
					"project_id": me.ProjectID,
				},
			},
			Points: []*monitoringpb.Point{
				{
					Interval: &monitoringpb.TimeInterval{
						EndTime: timestamppb.New(now),
					},
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
		// Merge common labels and distribution labels
		labels := make(map[string]string)
		for k, v := range me.CommonLabels {
			labels[k] = v
		}
		for k, v := range dist.Labels {
			labels[k] = v
		}

		metricType := "custom.googleapis.com/" + path.Join(me.MetricsNamePrefix, dist.Name)

		// Prepare bucket bounds
		bucketBounds := make([]float64, dist.NumBuckets+1)
		for i := 0; i <= dist.NumBuckets; i++ {
			bucketBounds[i] = float64(dist.Offset) + float64(dist.Step)*float64(i)
		}

		distBuckets := dist.GetAndClear()
		ts := &monitoringpb.TimeSeries{
			Metric: &metric.Metric{
				Type:   metricType,
				Labels: labels,
			},
			Resource: &monitoredres.MonitoredResource{
				Type: resourceType,
				Labels: map[string]string{
					"project_id": me.ProjectID,
				},
			},
			Points: []*monitoringpb.Point{
				{
					Interval: &monitoringpb.TimeInterval{
						EndTime: timestamppb.New(now),
					},
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_DistributionValue{
							DistributionValue: &distribution.Distribution{
								Count:                 distBuckets.NumSamples,
								Mean:                  distBuckets.Mean,
								SumOfSquaredDeviation: distBuckets.SumOfSquaredDeviation,
								BucketOptions: &distribution.Distribution_BucketOptions{
									Options: &distribution.Distribution_BucketOptions_ExplicitBuckets{
										ExplicitBuckets: &distribution.Distribution_BucketOptions_Explicit{
											Bounds: bucketBounds,
										},
									},
								},
								BucketCounts: distBuckets.Buckets,
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
func (me *MetricsEmitter) EmitEvery(interval time.Duration) *time.Ticker {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// Notify before emit listeners
			for _, listener := range me.BeforeEmitListeners {
				if listener != nil {
					listener()
				}
			}

			// Emit metrics
			me.Emit()
		}
	}()
	return ticker
}
