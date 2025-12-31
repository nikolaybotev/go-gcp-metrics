package gcpmetrics

import (
	"context"
	"io"
	"log"
	"maps"
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

// GcpMetricsEmitter handles the emission of metrics to Google Cloud Monitoring.
type GcpMetricsEmitter struct {
	Client            *monitoring.MetricClient
	ProjectID         string
	MonitoredResource *monitoredres.MonitoredResource
	MetricsNamePrefix string
	CommonLabels      map[string]string
	errorLogger       *log.Logger
	infoLogger        *log.Logger
}

// NewGcpMetricsEmitter creates a new GcpMetricsEmitter instance.
func NewGcpMetricsEmitter(
	client *monitoring.MetricClient,
	projectID string,
	monitoredResource *monitoredres.MonitoredResource,
	metricsNamePrefix string,
	opts *Options,
) *GcpMetricsEmitter {
	// Set defaults if nil
	if opts == nil {
		opts = &Options{}
	}
	if opts.ErrorLogger == nil {
		opts.ErrorLogger = log.Default()
	}
	if opts.InfoLogger == nil {
		opts.InfoLogger = log.New(io.Discard, "", 0)
	}
	if opts.CommonLabels == nil {
		opts.CommonLabels = make(map[string]string)
	}

	return &GcpMetricsEmitter{
		Client:            client,
		ProjectID:         projectID,
		MonitoredResource: monitoredResource,
		MetricsNamePrefix: metricsNamePrefix,
		CommonLabels:      opts.CommonLabels,
		errorLogger:       opts.ErrorLogger,
		infoLogger:        opts.InfoLogger,
	}
}

// mergeLabels merges common labels with metric-specific labels.
func (me *GcpMetricsEmitter) mergeLabels(specific map[string]string) map[string]string {
	labels := make(map[string]string, len(me.CommonLabels)+len(specific))
	maps.Copy(labels, me.CommonLabels)
	if specific != nil {
		maps.Copy(labels, specific)
	}
	return labels
}

// buildMetric constructs a metric.Metric with the correct type and merged labels.
func (me *GcpMetricsEmitter) buildMetric(name string, specificLabels map[string]string) *metric.Metric {
	return &metric.Metric{
		Type:   "custom.googleapis.com/" + path.Join(me.MetricsNamePrefix, name),
		Labels: me.mergeLabels(specificLabels),
	}
}

// Emit emits metrics from the provided Metrics to Google Cloud Monitoring.
func (me *GcpMetricsEmitter) Emit(ctx context.Context, metrics *Metrics) {
	if me.Client == nil {
		me.errorLogger.Println("Client must be set in GcpMetricsEmitter")
		return
	}
	if me.ProjectID == "" {
		me.errorLogger.Println("ProjectID must be set in GcpMetricsEmitter")
		return
	}
	if me.MonitoredResource == nil {
		me.errorLogger.Println("MonitoredResource must be set in GcpMetricsEmitter")
		return
	}

	now := time.Now()
	interval := &monitoringpb.TimeInterval{
		EndTime: timestamppb.New(now),
	}

	var timeSeriesList []*monitoringpb.TimeSeries

	// Emit counters
	for _, counter := range metrics.Counters {
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
	for _, gauge := range metrics.Gauges {
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
	for _, dist := range metrics.Distributions {
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
