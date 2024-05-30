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

type MyEvent struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type Item struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func HandleRequest(ctx context.Context, kinesisEvent events.KinesisEvent) error {
	// Create a new session
	sess := session.Must(session.NewSession(nil))

	// Create a DynamoDB service client
	svc := dynamodb.New(sess)

	for _, record := range kinesisEvent.Records {
		kinesisRecord := record.Kinesis

		// Decode the data from the Kinesis record into a MyEvent object
		var event MyEvent
		err := json.Unmarshal(kinesisRecord.Data, &event)
		if err != nil {
			return err
		}

		// Print the event ID and message to the CloudWatch log
		fmt.Printf("Processing event ID %s, message %s.\n", event.ID, event.Message)

		// Convert the MyEvent object to a DynamoDB attribute value
		av, err := dynamodbattribute.MarshalMap(Item{
			ID:      event.ID,
			Message: event.Message,
		})
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
