package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awskinesis"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type ServerlessDataProcessingPipelineStackProps struct {
	awscdk.StackProps
	HotDeploy       bool
	LambdasSrcPath  string
	LambdasDistPath string
}

func NewServerlessDataProcessingPipelineStack(scope constructs.Construct, id string, props *ServerlessDataProcessingPipelineStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Define the VPC
	vpc := awsec2.NewVpc(stack, jsii.String("VPC"), &awsec2.VpcProps{})

	// Create a Gateway VPC endpoint for DynamoDB
	awsec2.NewGatewayVpcEndpoint(stack, jsii.String("DynamoDbEndpoint"), &awsec2.GatewayVpcEndpointProps{
		Vpc:     vpc,
		Service: awsec2.GatewayVpcEndpointAwsService_DYNAMODB(),
	})

	// Create a VPC endpoint for Kinesis
	awsec2.NewInterfaceVpcEndpoint(stack, jsii.String("KinesisEndpoint"), &awsec2.InterfaceVpcEndpointProps{
		Vpc:     vpc,
		Service: awsec2.InterfaceVpcEndpointAwsService_KINESIS_STREAMS(),
	})

	// Define the Kinesis Stream
	stream := awskinesis.NewStream(stack, jsii.String("Kinesis"), &awskinesis.StreamProps{
		StreamName: jsii.String("KinesisStream"),
	})

	// Define the DynamoDB table
	table := awsdynamodb.NewTable(stack, jsii.String("DynamoDBTable"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		Stream: awsdynamodb.StreamViewType_NEW_IMAGE,
	})

	// Define the Lambda functions
	lambdaConfig := map[string]map[string]*string{
		"upstream": {
			"LAMBDA_STAGE:": jsii.String("upstream"),
			"STREAM_NAME":   stream.StreamName(),
		},
		"midstream": {
			"LAMBDA_STAGE:": jsii.String("midstream"),
			"TABLE_NAME":    table.TableName(),
		},
		"downstream": {
			"LAMBDA_STAGE:": jsii.String("downstream"),
		},
	}

	lambdas := make(map[string]awslambda.IFunction)
	lambdaBucket := awss3.Bucket_FromBucketName(stack, jsii.String("HotReloadingBucket"), jsii.String("hot-reload"))
	for k, v := range lambdaConfig {
		var lambdaCode awslambda.Code
		if props.HotDeploy {
			lambdaCode = awslambda.Code_FromBucket(lambdaBucket, jsii.String(filepath.Join(props.LambdasDistPath, k)), nil)
		} else {
			lambdaCode = awslambda.Code_FromAsset(jsii.String(filepath.Join(props.LambdasSrcPath, k)), &awss3assets.AssetOptions{
				Bundling: &awscdk.BundlingOptions{
					Image:   awscdk.DockerImage_FromRegistry(jsii.String("golang:1.21")),
					Command: &[]*string{jsii.String("bash"), jsii.String("-c"), jsii.String("go build -o /asset-output/main .")},
					Environment: &map[string]*string{
						"GOCACHE": jsii.String("/tmp/go-cache"),
						"GOOS":    jsii.String("linux"),
						"GOARCH":  jsii.String("amd64"),
					},
				},
			})
		}
		lambda := awslambda.NewFunction(stack, jsii.String("Lambda"+strings.ToTitle(k)), &awslambda.FunctionProps{
			Vpc:          vpc,
			Runtime:      awslambda.Runtime_GO_1_X(),
			Code:         lambdaCode,
			Handler:      jsii.String("main"),
			Environment:  &v,
			Architecture: awslambda.Architecture_X86_64(),
		})
		lambdas[k] = lambda
	}

	// Define the API Gateway
	api := awsapigateway.NewRestApi(stack, jsii.String("Api"), &awsapigateway.RestApiProps{
		DefaultIntegration: awsapigateway.NewLambdaIntegration(lambdas["upstream"], nil),
	})
	// Add Lambda function integration
	api.Root().AddMethod(jsii.String("POST"), awsapigateway.NewLambdaIntegration(lambdas["upstream"], nil), nil)

	// Connect the Lambda functions to the Kinesis Stream and DynamoDB Stream
	stream.GrantRead(lambdas["midstream"].Role())
	table.GrantStreamRead(lambdas["downstream"].Role())

	// Add the event sources to the Lambda functions
	lambdas["upstream"].AddEventSource(awslambdaeventsources.NewKinesisEventSource(stream, &awslambdaeventsources.KinesisEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
	}))
	lambdas["midstream"].AddEventSource(awslambdaeventsources.NewDynamoEventSource(table, &awslambdaeventsources.DynamoEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
	}))

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	// Get the current working directory
	rootDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	NewServerlessDataProcessingPipelineStack(app, "ServerlessDataProcessingPipelineStack", &ServerlessDataProcessingPipelineStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
		HotDeploy: os.Getenv("HOT_DEPLOY") == "true",
		LambdasDistPath: func() string {
			lambdaDistPath := os.Getenv("LAMBDA_DIST_PATH")
			if lambdaDistPath == "" {
				lambdaDistPath = "lambda/dist"
			}
			lambdaDistPath = filepath.Join(rootDirectory, lambdaDistPath)
			return lambdaDistPath
		}(),
		LambdasSrcPath: func() string {
			lambdaSrcPath := os.Getenv("LAMBDA_SRC_PATH")
			if lambdaSrcPath == "" {
				lambdaSrcPath = "lambda/src"
			}
			lambdaSrcPath = filepath.Join(rootDirectory, lambdaSrcPath)
			return lambdaSrcPath
		}(),
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
