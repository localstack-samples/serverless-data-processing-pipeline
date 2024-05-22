package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type MyEvent struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func HandleRequest(ctx context.Context, event MyEvent) (string, error) {
	// Create a new session
	sess := session.Must(session.NewSession(nil))

	// Create a Kinesis service client
	svc := kinesis.New(sess)

	// Convert the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	// Put record to Kinesis Stream
	_, err = svc.PutRecord(&kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(os.Getenv("STREAM_NAME")),
		PartitionKey: aws.String(event.ID),
	})

	if err != nil {
		return "", err
	}

	return "Success", nil
}

func main() {
	lambda.Start(HandleRequest)
}
