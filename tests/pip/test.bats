#!/usr/bin/env bats

export LAMBDA_ROLE="arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
export AWS_ACCOUNT_ID="$(aws sts get-caller-identity | jq -r ".Account")"
export LAMBDA_FUNCTION_NAME=lambda-python39-pip
export LAMBDA_RUNTIME=python3.9
export LAMBDA_HANDLER=function.handler

setup() {
  aws lambda delete-function --function-name "$LAMBDA_FUNCTION_NAME" 2>/dev/null || true
  aws iam detach-role-policy --role-name "$LAMBDA_FUNCTION_NAME" --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole 2>/dev/null || true
  aws iam delete-role --role-name "$LAMBDA_FUNCTION_NAME" 2>/dev/null || true
}

teardown() {
  aws lambda delete-function --function-name "$LAMBDA_FUNCTION_NAME" 2>/dev/null || true
  aws iam detach-role-policy --role-name "$LAMBDA_FUNCTION_NAME" --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole 2>/dev/null || true
  aws iam delete-role --role-name "$LAMBDA_FUNCTION_NAME" 2>/dev/null || true
}

@test "aws test" {
  run /bin/bash -c "lambda-builder build"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "aws iam create-role --role-name '$LAMBDA_FUNCTION_NAME' --tags 'Key=app,Value=lambda-builder' --tags 'Key=com.dokku.lambda-builder/runtime,Value=$LAMBDA_RUNTIME'  --assume-role-policy-document '{\"Version\": \"2012-10-17\", \"Statement\": [{ \"Effect\": \"Allow\", \"Principal\": {\"Service\": \"lambda.amazonaws.com\"}, \"Action\": \"sts:AssumeRole\"}]}'"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "aws iam attach-role-policy --role-name '$LAMBDA_FUNCTION_NAME' --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "sleep 10"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "aws lambda create-function --function-name '$LAMBDA_FUNCTION_NAME' --package-type Zip --tags 'app=lambda-builder,com.dokku.lambda-builder/runtime=$LAMBDA_RUNTIME' --role 'arn:aws:iam::${AWS_ACCOUNT_ID}:role/$LAMBDA_FUNCTION_NAME' --zip-file fileb://lambda.zip --runtime '$LAMBDA_RUNTIME' --handler '$LAMBDA_HANDLER'"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "sleep 10"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "aws lambda get-function --function-name '$LAMBDA_FUNCTION_NAME'"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]

  run /bin/bash -c "aws lambda invoke --cli-binary-format raw-in-base64-out --function-name '$LAMBDA_FUNCTION_NAME' --payload '{\"name\": \"World\"}' response.json"
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}