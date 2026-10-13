export AWS_ACCESS_KEY_ID ?= test
export AWS_SECRET_ACCESS_KEY ?= test
export AWS_DEFAULT_REGION=us-east-1
SHELL := /bin/bash

usage:		## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$//' | sed -e 's/##//'

install:	## Install dependencies
	@which lstk || npm install -g @localstack/lstk
	@which cdk || npm install -g aws-cdk

start:		## Start LocalStack
	@test -n "${LOCALSTACK_AUTH_TOKEN}" || (echo "LOCALSTACK_AUTH_TOKEN is not set. Find your token at https://app.localstack.cloud/workspace/auth-token"; exit 1)
	@LOCALSTACK_AUTH_TOKEN=$(LOCALSTACK_AUTH_TOKEN) lstk start

build:		## Build the Lambda binaries used by hot reloading into lambda/dist
	@for fn in upstream midstream downstream; do \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -C ./lambda/src/$$fn -o ../../dist/$$fn/bootstrap || exit 1; \
		echo "Built lambda/dist/$$fn/bootstrap"; \
	done

deploy:		## Deploy the stack to LocalStack with hot reloading
	@USE_LOCALSTACK=true HOT_DEPLOY=true lstk cdk bootstrap
	@USE_LOCALSTACK=true HOT_DEPLOY=true lstk cdk deploy --require-approval never

test:		## Send a request through the pipeline and check every stage handled it
	@./test.sh

stop:		## Stop LocalStack
	@lstk stop

logs:		## Save the logs in a separate file
	@lstk logs > logs.txt

.PHONY: usage install start build deploy test stop logs
