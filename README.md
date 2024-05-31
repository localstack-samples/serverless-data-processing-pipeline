# serverless-data-processing-pipeline

This is a sample CDK app that creates a *API Gateway -> Lambda -> Kinesis Stream -> Lambda -> DynamoDB -> DynamoDB stream -> Lambda -> Postgres* chain and then we benchmark the time it takes to complete this loop on a M1 max chip.

## Prerequisites

The following dependencies need to be available on your machine:

1. [Go](https://go.dev/doc/install).

1. [Localstack CLI](https://docs.localstack.cloud/getting-started/installation/).

1. [CDK CLI](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html).

1. [Watchman](https://facebook.github.io/watchman/docs/install).

1. [jq](https://jqlang.github.io/jq/download/).

## Commands

 * `localstack start`                         start LocalStack with the Docker executor
 * `cdk bootstrap`                            bootstrap cdk stack onto AWS/LocalStack
 * `cdk deploy`                               deploy this stack to your default AWS account/region
 * `cdk diff`                                 compare deployed stack with current state
 * `cdk synth`                                emits the synthesized CloudFormation template
 * `go test`                                  run unit tests
 * `watchman [upstream|midstream|downstream]` watch and hot-reload lambda functions

## Configuration

* `USE_LOCALSTACK`   set to `true` if the stack is deployed to LocalStack
* `HOT_DEPLOY`       set to `true` if the hot-reloading feature is to be enabled
* `LAMBDA_DIST_PATH` directory where binaries for the hot-reloading feature are stored (optional)
* `LAMBDA_SRC_PATH`  directory where the src of the lambda functions is found

## Deploy

On LocalStack:

```bash
export USE_LOCALSTACK=true
export HOT_DEPLOY=true
cdklocal bootstrap
cdklocal deploy --require-approval=never
```

On AWS:
```
cdk bootstrap --profile aws
cdk deploy --require-approval=never --profile aws
```

## Sample Run

After deploying the stack, retrieve the method's endpoint by inspecting the CfnOutput outputs like in the following example:

```sh
localstack@macintosh serverless-data-processing-pipeline % USE_LOCALSTACK=true HOT_DEPLOY=true cdklocal deploy --require-approval=never                                                   

✨  Synthesis time: 3.72s

ServerlessDataProcessingPipelineStack:  start: Building dd5711540f04e06aa955d7f4862fc04e8cdea464cb590dae91ed2976bb78098e:current_account-current_region
ServerlessDataProcessingPipelineStack:  success: Built dd5711540f04e06aa955d7f4862fc04e8cdea464cb590dae91ed2976bb78098e:current_account-current_region
ServerlessDataProcessingPipelineStack:  start: Building 4c4836f6c768f4500c058ac6a02f2090830a58eb1a0e58d59a5c7ffadf208861:current_account-current_region
ServerlessDataProcessingPipelineStack:  success: Built 4c4836f6c768f4500c058ac6a02f2090830a58eb1a0e58d59a5c7ffadf208861:current_account-current_region
ServerlessDataProcessingPipelineStack:  start: Publishing dd5711540f04e06aa955d7f4862fc04e8cdea464cb590dae91ed2976bb78098e:current_account-current_region
ServerlessDataProcessingPipelineStack:  start: Publishing 4c4836f6c768f4500c058ac6a02f2090830a58eb1a0e58d59a5c7ffadf208861:current_account-current_region
ServerlessDataProcessingPipelineStack:  success: Published 4c4836f6c768f4500c058ac6a02f2090830a58eb1a0e58d59a5c7ffadf208861:current_account-current_region
ServerlessDataProcessingPipelineStack:  success: Published dd5711540f04e06aa955d7f4862fc04e8cdea464cb590dae91ed2976bb78098e:current_account-current_region
ServerlessDataProcessingPipelineStack: deploying... [1/1]
ServerlessDataProcessingPipelineStack: creating CloudFormation changeset...

 ✅  ServerlessDataProcessingPipelineStack

✨  Deployment time: 30.69s

Outputs:
ServerlessDataProcessingPipelineStack.ApiEndpoint4F160690 = https://tsyeuri986.execute-api.localhost.localstack.cloud:4566/prod/
ServerlessDataProcessingPipelineStack.ApiGatewayMethodEndpoint = https://tsyeuri986.execute-api.localhost.localstack.cloud:4566/prod/
ServerlessDataProcessingPipelineStack.DynamoDBTableName = ServerlessDataProcessingPipeline-DynamoDBTable59784FC0-072648f2
ServerlessDataProcessingPipelineStack.Environment = LocalStack
ServerlessDataProcessingPipelineStack.KinesisStreamName = KinesisStream
Stack ARN:
arn:aws:cloudformation:us-east-1:000000000000:stack/ServerlessDataProcessingPipelineStack/68a8d688

✨  Total time: 34.4s

localstack@macintosh serverless-data-processing-pipeline % export API_ENDPOINT="https://tsyeuri986.execute-api.localhost.localstack.cloud:4566/prod/"
```

Followed by a sample request:

```sh
localstack@macintosh serverless-data-processing-pipeline % timestamp=$(awk 'BEGIN {srand(); print srand()}')
localstack@macintosh serverless-data-processing-pipeline % curl -XPOST -H "Content-Type: application/json" $API_ENDPOINT -d "$(jq -n --arg ts "$timestamp" '{id: "1", message: "Hello World", timestamp: $ts | tonumber}')" -i
HTTP/2 200 
content-type: application/json
content-length: 21
date: Fri, 31 May 2024 18:07:54 GMT
server: hypercorn-h2

{"message":"success"}
```
