package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awskinesis"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type ServerlessDataProcessingPipelineStackProps struct {
	awscdk.StackProps
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
	gateway := awsec2.NewGatewayVpcEndpoint(stack, jsii.String("DynamoDbEndpoint"), &awsec2.GatewayVpcEndpointProps{
		Vpc:     vpc,
		Service: awsec2.GatewayVpcEndpointAwsService_DYNAMODB(),
	})

	// Add a route to the main route table that points to the DynamoDB Gateway Endpoint
	publicSubnets := vpc.PublicSubnets()
	if publicSubnets != nil {
		if len(*publicSubnets) > 0 {
			routeTable := (*publicSubnets)[0].RouteTable()

			// Add a route to the main route table that points to the DynamoDB Gateway Endpoint
			awsec2.NewCfnRoute(stack, jsii.String("DynamoDbRoute"), &awsec2.CfnRouteProps{
				RouteTableId:         routeTable.RouteTableId(),
				DestinationCidrBlock: jsii.String("0.0.0.0/0"),
				GatewayId:            gateway.VpcEndpointId(),
			})
		}
	}

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
	lambdaUpstream := awslambda.NewFunction(stack, jsii.String("LambdaUpstream"), &awslambda.FunctionProps{
		Vpc:     vpc,
		Runtime: awslambda.Runtime_GO_1_X(),
		Code: awslambda.Code_FromAsset(jsii.String("lambda/upstream"), &awss3assets.AssetOptions{
			Bundling: &awscdk.BundlingOptions{
				Image:   awscdk.DockerImage_FromRegistry(jsii.String("public.ecr.aws/sam/build-go1.x")),
				Command: &[]*string{jsii.String("go"), jsii.String("build"), jsii.String("-o"), jsii.String("/asset-output/main"), jsii.String(".")},
			},
		}),
		Handler: jsii.String("main"),
		Environment: &map[string]*string{
			"LAMBDA_STAGE": jsii.String("upstream"),
			"STREAM_NAME":  stream.StreamName(),
		},
	})
	lambdaMidstream := awslambda.NewFunction(stack, jsii.String("LambdaMidstream"), &awslambda.FunctionProps{
		Vpc:     vpc,
		Runtime: awslambda.Runtime_GO_1_X(),
		Code: awslambda.Code_FromAsset(jsii.String("lambda/midstream"), &awss3assets.AssetOptions{
			Bundling: &awscdk.BundlingOptions{
				Image:   awscdk.DockerImage_FromRegistry(jsii.String("public.ecr.aws/sam/build-go1.x")),
				Command: &[]*string{jsii.String("go"), jsii.String("build"), jsii.String("-o"), jsii.String("/asset-output/main"), jsii.String(".")},
			},
		}),
		Handler: jsii.String("main"),
		Environment: &map[string]*string{
			"LAMBDA_STAGE": jsii.String("midstream"),
			"TABLE_NAME":   table.TableName(),
		},
	})
	lambdaDownstream := awslambda.NewFunction(stack, jsii.String("LambdaDownstream"), &awslambda.FunctionProps{
		Vpc:     vpc,
		Runtime: awslambda.Runtime_GO_1_X(),
		Code: awslambda.Code_FromAsset(jsii.String("lambda/downstream"), &awss3assets.AssetOptions{
			Bundling: &awscdk.BundlingOptions{
				Image:   awscdk.DockerImage_FromRegistry(jsii.String("public.ecr.aws/sam/build-go1.x")),
				Command: &[]*string{jsii.String("go"), jsii.String("build"), jsii.String("-o"), jsii.String("/asset-output/main"), jsii.String(".")},
			},
		}),
		Handler: jsii.String("main"),
		Environment: &map[string]*string{
			"LAMBDA_STAGE": jsii.String("downstream"),
		},
	})

	// Define the API Gateway
	api := awsapigateway.NewRestApi(stack, jsii.String("Api"), &awsapigateway.RestApiProps{
		DefaultIntegration: awsapigateway.NewLambdaIntegration(lambdaUpstream, nil),
	})
	// Add Lambda function integration
	api.Root().AddMethod(jsii.String("POST"), awsapigateway.NewLambdaIntegration(lambdaUpstream, nil), nil)

	// Connect the Lambda functions to the Kinesis Stream and DynamoDB Stream
	stream.GrantRead(lambdaMidstream.Role())
	table.GrantStreamRead(lambdaDownstream.Role())

	// Add the event sources to the Lambda functions
	lambdaUpstream.AddEventSource(awslambdaeventsources.NewKinesisEventSource(stream, &awslambdaeventsources.KinesisEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
	}))
	lambdaMidstream.AddEventSource(awslambdaeventsources.NewDynamoEventSource(table, &awslambdaeventsources.DynamoEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
	}))

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewServerlessDataProcessingPipelineStack(app, "ServerlessDataProcessingPipelineStack", &ServerlessDataProcessingPipelineStackProps{
		awscdk.StackProps{
			Env: env(),
		},
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
