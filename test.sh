#!/bin/bash
# Send one request through the pipeline and check every stage handled it:
# API Gateway -> upstream Lambda -> Kinesis -> midstream Lambda -> DynamoDB
# -> DynamoDB stream -> downstream Lambda -> CloudWatch metric
set -euo pipefail

STACK=ServerlessDataProcessingPipelineStack
MAX_ATTEMPTS=${MAX_ATTEMPTS:-60}
ID="ci-$(date +%s)"
MESSAGE="Hello from the pipeline test"

output() {
  lstk aws cloudformation describe-stacks --stack-name "$STACK" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue" --output text
}

function_name() {
  lstk aws cloudformation describe-stack-resources --stack-name "$STACK" \
    --query "StackResources[?ResourceType=='AWS::Lambda::Function' && starts_with(LogicalResourceId, 'Lambda$1')].PhysicalResourceId" \
    --output text
}

# post <body>: print the response body, then the HTTP status on the last line
post() {
  curl -sk -m 60 -w '\n%{http_code}' -X POST -H 'Content-Type: application/json' "$ENDPOINT" -d "$1"
}

ENDPOINT=$(output ApiGatewayMethodEndpoint)
TABLE=$(output DynamoDBTableName)
DOWNSTREAM=$(function_name DOWNSTREAM)
echo "API: $ENDPOINT"
echo "Table: $TABLE"

# The event source mappings start at LATEST, so records sent before they poll are skipped
for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
  states=$(lstk aws lambda list-event-source-mappings --query 'EventSourceMappings[].State' --output text)
  [ "$(echo "$states" | tr '\t' '\n' | grep -c '^Enabled$')" = 2 ] && break
  echo "Attempt $attempt/$MAX_ATTEMPTS: event source mappings are '$states'"
  sleep 2
done
sleep 5

response=$(post '{"id": "incomplete"}')
status=${response##*$'\n'}
if [ "$status" != 400 ] || [[ "$response" != *"message is required"* ]]; then
  echo "Test failed: an item without a message returned HTTP $status: ${response%$'\n'*}"
  exit 1
fi
echo "PASS: an item without a message is rejected with HTTP 400"

START_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
response=$(post "$(jq -n --arg id "$ID" --arg m "$MESSAGE" --argjson ts "$(date -u +%s)" '{id: $id, message: $m, timestamp: $ts}')")
status=${response##*$'\n'}
body=${response%$'\n'*}
if [ "$status" != 200 ] || ! echo "$body" | jq -e '.message == "success"' > /dev/null; then
  echo "Test failed: POST returned HTTP $status: $body"
  exit 1
fi
echo "PASS: API accepted item $ID"

stored=""
for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
  stored=$(lstk aws dynamodb get-item --table-name "$TABLE" --key "{\"id\": {\"S\": \"$ID\"}}" \
    --query Item.message.S --output text)
  [ "$stored" = "$MESSAGE" ] && break
  echo "Attempt $attempt/$MAX_ATTEMPTS: item not in DynamoDB yet"
  sleep 2
done
if [ "$stored" != "$MESSAGE" ]; then
  echo "Test failed: the midstream Lambda did not write item $ID to $TABLE"
  exit 1
fi
echo "PASS: the midstream Lambda wrote the item from Kinesis to DynamoDB"

logged=""
for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
  logged=$(lstk aws logs filter-log-events --log-group-name "/aws/lambda/$DOWNSTREAM" \
    --query 'events[].message' --output text 2>/dev/null || true)
  [[ "$logged" == *"New {ID: $ID, Message: $MESSAGE"* ]] && break
  echo "Attempt $attempt/$MAX_ATTEMPTS: downstream Lambda has not logged the item yet"
  sleep 2
done
if [[ "$logged" != *"New {ID: $ID, Message: $MESSAGE"* ]]; then
  echo "Test failed: the downstream Lambda did not receive item $ID from the DynamoDB stream"
  exit 1
fi
echo "PASS: the downstream Lambda received the item from the DynamoDB stream"

samples=0
for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
  samples=$(lstk aws cloudwatch get-metric-statistics \
    --namespace ServerlessDataProcessingPipeline/Latencies --metric-name Latency \
    --start-time "$START_TIME" --end-time "$(date -u -v+1M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+1 minute' +%Y-%m-%dT%H:%M:%SZ)" \
    --period 60 --statistics SampleCount --output json | jq '[.Datapoints[].SampleCount] | add // 0')
  [ "${samples%.*}" -ge 1 ] && break
  echo "Attempt $attempt/$MAX_ATTEMPTS: no latency datapoint yet"
  sleep 2
done
if [ "${samples%.*}" -lt 1 ]; then
  echo "Test failed: no Latency datapoint in CloudWatch since $START_TIME"
  exit 1
fi
echo "PASS: the downstream Lambda recorded the pipeline latency in CloudWatch"

echo "Test passed: the item flowed through API Gateway, Kinesis, DynamoDB and the DynamoDB stream"
