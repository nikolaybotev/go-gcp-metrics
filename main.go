package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT env var must be set")
	}

	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		log.Fatalf("failed to create metric client: %v", err)
	}
	defer client.Close()

	metricType := "custom.googleapis.com/sample_counter"
	resourceType := "global"

	for {
		now := time.Now()
		value := rand.Int63n(100)

		req := &monitoringpb.CreateTimeSeriesRequest{
			Name: "projects/" + projectID,
			TimeSeries: []*monitoringpb.TimeSeries{
				{
					Metric: &metric.Metric{
						Type: metricType,
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
			log.Printf("failed to write time series data: %v", err)
		} else {
			fmt.Printf("Published value %d at %s\n", value, now.Format(time.RFC3339))
		}

		time.Sleep(10 * time.Second)
	}
}
