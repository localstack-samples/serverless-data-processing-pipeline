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

type Item struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type MyResponse struct {
	Message string `json:"message"`
}

func (item *Item) UnmarshalJSON(data []byte) error {
	type Alias Item
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(item),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if item.ID == "" {
		return fmt.Errorf("id is required")
	}
	if item.Message == "" {
		return fmt.Errorf("message is required")
	}
	if item.Timestamp == 0 {
		return fmt.Errorf("timestamp is required")
	}
	return nil
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
	var item Item
	err := json.Unmarshal([]byte(request.Body), &item)
	if err != nil {
		return restResponse(400, fmt.Sprintf("failed to unmarshal request body: %s", err))
	}

	fmt.Println("Received item: ", item.ID, item.Message)

	// Create a new session
	var endpointUrl *string
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		endpointUrl = aws.String(os.Getenv("AWS_ENDPOINT_URL"))
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint: endpointUrl,
	}))

	// Create a Kinesis service client
	svc := kinesis.New(sess)

	// Convert the event to JSON
	data, err := json.Marshal(item)
	if err != nil {
		return restResponse(500, fmt.Sprintf("failed to marshal payload: %s", err))
	}

	// Put record to Kinesis Stream
	_, err = svc.PutRecord(&kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(os.Getenv("STREAM_NAME")),
		PartitionKey: aws.String(item.ID),
	})
	if err != nil {
		return restResponse(500, fmt.Sprintf("failed to put record to %s Kinesis Stream: %s", os.Getenv("STREAM_NAME"), err))
	}

	return restResponse(200, "success")
}

func main() {
	lambda.Start(HandleRequest)
}
