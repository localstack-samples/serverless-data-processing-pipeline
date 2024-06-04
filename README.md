# serverless-data-processing-pipeline

This is a sample CDK app that creates a *API Gateway -> Lambda -> Kinesis Stream -> Lambda -> DynamoDB -> DynamoDB stream -> Lambda -> Postgres* chain and then we benchmark the time it takes to complete this loop on a M1 max chip.

## Prerequisites

The following dependencies need to be available on your machine:

1. [Go](https://go.dev/doc/install).

1. [Localstack CLI](https://docs.localstack.cloud/getting-started/installation/).

1. [CDK CLI](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html).

1. [Watchman](https://facebook.github.io/watchman/docs/install).

1. [jq](https://jqlang.github.io/jq/download/).

1. [k6](https://k6.io/docs/get-started/installation/).

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

## Stress Test

```sh
$ k6 run -e API_ENDPOINT=$API_ENDPOINT loadtest.js


          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

     execution: local
        script: loadtest.js
        output: -

     scenarios: (100.00%) 1 scenario, 10 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 10 looping VUs for 1m0s (gracefulStop: 30s)


     ✓ status was 200
     ✓ transaction time OK

     checks.........................: 100.00% ✓ 3432      ✗ 0   
     data_received..................: 272 kB  4.5 kB/s
     data_sent......................: 235 kB  3.9 kB/s
     http_req_blocked...............: avg=484.25µs min=0s       med=1µs      max=87.94ms  p(90)=1µs      p(95)=1µs     
     http_req_connecting............: avg=4.97µs   min=0s       med=0s       max=940µs    p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=350.92ms min=203.55ms med=339.86ms max=725.44ms p(90)=406.8ms  p(95)=488.96ms
       { expected_response:true }...: avg=350.92ms min=203.55ms med=339.86ms max=725.44ms p(90)=406.8ms  p(95)=488.96ms
     http_req_failed................: 0.00%   ✓ 0         ✗ 1716
     http_req_receiving.............: avg=41.04ms  min=31.15ms  med=40.83ms  max=55.17ms  p(90)=42.03ms  p(95)=42.94ms 
     http_req_sending...............: avg=64.99µs  min=12µs     med=42µs     max=2.06ms   p(90)=114.5µs  p(95)=150µs   
     http_req_tls_handshaking.......: avg=213.15µs min=0s       med=0s       max=41.6ms   p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=309.81ms min=162.9ms  med=298.6ms  max=689.61ms p(90)=366.07ms p(95)=441.45ms
     http_reqs......................: 1716    28.289592/s
     iteration_duration.............: avg=351.62ms min=225.77ms med=340.5ms  max=725.59ms p(90)=406.92ms p(95)=489.08ms
     iterations.....................: 1716    28.289592/s
     vus............................: 10      min=10      max=10
     vus_max........................: 10      min=10      max=10


running (1m00.7s), 00/10 VUs, 1716 complete and 0 interrupted iterations
default ✓ [======================================] 10 VUs  1m0s
```

And then let's wait until all requests have been processed by the `midstream` and `downstream` Lambda functions. Let's also save the timestamps that indicate how much time it took each request to flow through the entire pipeline.

```sh
$ ./wait_requests timestamps.json
Monitoring CloudWatch metrics for new datapoints...
No new datapoints added. Exiting.
Exporting CloudWatch metrics to timestamps.json...
```