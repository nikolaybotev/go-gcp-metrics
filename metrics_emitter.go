package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MetricsEmitter struct {
	Counters []*Counter
}

func NewMetricsEmitter() *MetricsEmitter {
	return &MetricsEmitter{
		Counters: []*Counter{},
	}
}

func (me *MetricsEmitter) AddCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

func (me *MetricsEmitter) Emit() {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Println("GOOGLE_CLOUD_PROJECT env var must be set")
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

		req := &monitoringpb.CreateTimeSeriesRequest{
			Name: "projects/" + projectID,
			TimeSeries: []*monitoringpb.TimeSeries{
				{
					Metric: &metric.Metric{
						Type:   "custom.googleapis.com/" + counter.Name,
						Labels: counter.Labels,
					},
					Resource: &monitoredres.MonitoredResource{
						Type: resourceType,
						Labels: map[string]string{
							"project_id": projectID,
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
