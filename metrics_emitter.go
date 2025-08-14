package main

import (
	"context"
	"fmt"
	"log"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MetricsEmitter struct {
	ProjectID         string
	MetricsNamePrefix string
	CommonLabels      map[string]string
	Counters          []*Counter
}

func NewMetricsEmitter(projectID string, metricsNamePrefix string, commonLabels map[string]string) *MetricsEmitter {
	return &MetricsEmitter{
		ProjectID:         projectID,
		MetricsNamePrefix: metricsNamePrefix,
		CommonLabels:      commonLabels,
		Counters:          []*Counter{},
	}
}

func (me *MetricsEmitter) AddCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

func (me *MetricsEmitter) Emit() {
	if me.ProjectID == "" {
		log.Println("ProjectID must be set in MetricsEmitter")
		return
	}

	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		log.Printf("failed to create metric client: %v", err)
		return
	}
	defer client.Close()

	resourceType := "global"

	for _, counter := range me.Counters {
		now := time.Now()
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

		req := &monitoringpb.CreateTimeSeriesRequest{
			Name: "projects/" + me.ProjectID,
			TimeSeries: []*monitoringpb.TimeSeries{
				{
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
				},
			},
		}

		if err := client.CreateTimeSeries(ctx, req); err != nil {
			log.Printf("failed to write time series data for %s: %v", counter.Name, err)
		} else {
			fmt.Printf("Published counter %s value %d at %s\n", counter.Name, value, now.Format(time.RFC3339))
		}
	}
}
