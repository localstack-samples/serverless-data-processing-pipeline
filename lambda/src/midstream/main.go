package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type Item struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func HandleRequest(ctx context.Context, kinesisEvent events.KinesisEvent) error {
	// Create a new session
	var endpointUrl *string
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		endpointUrl = aws.String(os.Getenv("AWS_ENDPOINT_URL"))
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint: endpointUrl,
	}))

	// Create a DynamoDB service client
	svc := dynamodb.New(sess)

	for _, record := range kinesisEvent.Records {
		kinesisRecord := record.Kinesis

		// Decode the data from the Kinesis record into a MyEvent object
		var item Item
		err := json.Unmarshal(kinesisRecord.Data, &item)
		if err != nil {
			return err
		}

		// Print the event ID and message to the CloudWatch log
		fmt.Printf("Processing item ID %s, message %s.\n", item.ID, item.Message)

		// Convert the MyEvent object to a DynamoDB attribute value
		av, err := dynamodbattribute.MarshalMap(item)
		if err != nil {
			return err
		}

		// Put the item into the DynamoDB table
		_, err = svc.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(os.Getenv("TABLE_NAME")),
			Item:      av,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
