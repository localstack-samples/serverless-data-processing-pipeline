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
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type MyEvent struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type MyResponse struct {
	Message string `json:"message"`
}

func restResponse(status int, message string) (events.APIGatewayProxyResponse, error) {
	myResponse := MyResponse{
		Message: message,
	}
	body, err := json.Marshal(myResponse)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("failed to marshal response body: %s", err),
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var event MyEvent
	err := json.Unmarshal([]byte(request.Body), &event)
	if err != nil {
		return restResponse(400, fmt.Sprintf("failed to unmarshal request body: %s", err))
	}

	fmt.Println("Received event: ", event.ID, event.Message)

	// Create a new session
	sess := session.Must(session.NewSession(nil))

	// Create a Kinesis service client
	svc := kinesis.New(sess)

	// Convert the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return restResponse(500, fmt.Sprintf("failed to marshal payload: %s", err))
	}

	// Put record to Kinesis Stream
	_, err = svc.PutRecord(&kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(os.Getenv("STREAM_NAME")),
		PartitionKey: aws.String(event.ID),
	})
	if err != nil {
		return restResponse(500, fmt.Sprintf("failed to put record to %s Kinesis Stream: %s", os.Getenv("STREAM_NAME"), err))
	}

	return restResponse(200, "success")
}

func main() {
	lambda.Start(HandleRequest)
}
