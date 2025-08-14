package main

import (
	"context"
	"fmt"
	"log"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MetricsEmitter struct {
	client            *monitoring.MetricClient
	ProjectID         string
	MetricsNamePrefix string
	CommonLabels      map[string]string
	Counters          []*Counter
}

func NewMetricsEmitter(
	client *monitoring.MetricClient,
	projectID string,
	metricsNamePrefix string,
	commonLabels map[string]string,
) *MetricsEmitter {
	return &MetricsEmitter{
		client:            client,
		ProjectID:         projectID,
		MetricsNamePrefix: metricsNamePrefix,
		CommonLabels:      commonLabels,
		Counters:          []*Counter{},
	}
}

func (me *MetricsEmitter) AddCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

func (me *MetricsEmitter) Counter(name string, labels map[string]string) *Counter {
	counter := NewCounterWithLabels(name, labels)
	me.AddCounter(counter)
	return counter
}

func (me *MetricsEmitter) Emit() {
	if me.ProjectID == "" {
		log.Println("ProjectID must be set in MetricsEmitter")
		return
	}
	if me.client == nil {
		log.Println("Metric client is not initialized")
		return
	}

	ctx := context.Background()

	resourceType := "global"
	now := time.Now()

	var timeSeriesList []*monitoringpb.TimeSeries

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

		metricType := "custom.googleapis.com/" + me.MetricsNamePrefix + counter.Name

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

	if len(timeSeriesList) == 0 {
		return
	}

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + me.ProjectID,
		TimeSeries: timeSeriesList,
	}

	if err := me.client.CreateTimeSeries(ctx, req); err != nil {
		log.Printf("failed to write time series data: %v", err)
	} else {
		for _, counter := range me.Counters {
			fmt.Printf("Published counter %s value %d at %s\n", counter.Name, counter.Value(), now.Format(time.RFC3339))
		}
	}
}

// EmitEvery schedules Emit to run at the given interval in a new goroutine.
func (me *MetricsEmitter) EmitEvery(interval time.Duration) *time.Ticker {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			me.Emit()
		}
	}()
	return ticker
}
