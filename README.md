# serverless-data-processing-pipeline

This is a sample CDK app that creates a *API Gateway -> Lambda -> Kinesis Stream -> Lambda -> DynamoDB -> DynamoDB stream -> Lambda -> Postgres* chain and then we benchmark the time it takes to complete this loop on a M1 max chip.

## Prerequisites

The following dependencies need to be available on your machine:

1. [Go](https://go.dev/doc/install).

1. [Localstack CLI](https://docs.localstack.cloud/getting-started/installation/).

1. [CDK CLI](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html).

## Useful commands

 * `cdk deploy`      deploy this stack to your default AWS account/region
 * `cdk diff`        compare deployed stack with current state
 * `cdk synth`       emits the synthesized CloudFormation template
 * `go test`         run unit tests
