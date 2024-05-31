package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type Item struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func (item Item) String() string {
	return fmt.Sprintf("{ID: %s, Message: %s, Timestamp: %d}", item.ID, item.Message, item.Timestamp)
}

func extractItem(image map[string]events.DynamoDBAttributeValue) (Item, error) {
	var item Item
	var err error

	item.ID = image["id"].String()
	item.Message = image["message"].String()

	item.Timestamp, err = image["timestamp"].Integer()
	if err != nil {
		return item, err
	}

	return item, nil
}

func recordLatency(cwm *cloudwatch.CloudWatch, item Item) error {
	// Compute the latency of the entire processing pipeline
	currentTimestamp := time.Now().UTC().Unix()
	oldTimestamp := item.Timestamp
	totalLatency := currentTimestamp - oldTimestamp

	// Record the latency in CloudWatch
	_, err := cwm.PutMetricData(
		&cloudwatch.PutMetricDataInput{
			Namespace: aws.String("ServerlessDataProcessingPipeline/Latencies"),
			MetricData: []*cloudwatch.MetricDatum{
				{
					MetricName: aws.String("Latency"),
					Unit:       aws.String("Count"),
					Value:      aws.Float64(float64(totalLatency)),
				},
			},
		},
	)
	return err
}

func handleRequest(ctx context.Context, dynamodbEvent events.DynamoDBEvent) {
	// Create a new session
	var endpointUrl *string
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		endpointUrl = aws.String(os.Getenv("AWS_ENDPOINT_URL"))
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint: endpointUrl,
	}))

	// Create a CloudWatch service client
	cwm := cloudwatch.New(sess)

	for _, record := range dynamodbEvent.Records {
		// Print new image (the new version of the item) to stdout
		if image := record.Change.NewImage; image != nil {
			item, err := extractItem(image)
			if err != nil {
				fmt.Printf("Error unmarshalling item: %v\n", err)
				return
			}
			fmt.Printf("New %s\n", item)

			// Record the latency of the item processing
			err = recordLatency(cwm, item)
			if err != nil {
				fmt.Printf("Error recording latency: %v\n", err)
			}
		}

		// Print old image (the previous version of the item) to stdout
		if image := record.Change.OldImage; image != nil {
			item, err := extractItem(image)
			if err != nil {
				fmt.Printf("Error unmarshalling item: %v\n", err)
				return
			}
			fmt.Printf("Old %s\n", item)
		}
	}
}

func main() {
	lambda.Start(handleRequest)
}
