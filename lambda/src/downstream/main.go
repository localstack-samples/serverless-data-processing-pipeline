package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(ctx context.Context, dynamodbEvent events.DynamoDBEvent) {
	for _, record := range dynamodbEvent.Records {
		fmt.Printf("Processing request data for event ID %s, type %s.\n", record.EventID, record.EventName)

		// Print new image (the new version of the item) to stdout
		if image := record.Change.NewImage; image != nil {
			fmt.Printf("New item: %v\n", image)
		}

		// Print old image (the previous version of the item) to stdout
		if image := record.Change.OldImage; image != nil {
			fmt.Printf("Old item: %v\n", image)
		}
	}
}

func main() {
	lambda.Start(handleRequest)
}
