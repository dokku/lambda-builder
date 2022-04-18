#!/usr/bin/env bats

export SYSTEM_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
export LAMBDA_BUILDER_BIN="build/$SYSTEM_NAME/lambda-builder-amd64"

setup_file() {
  make prebuild "$LAMBDA_BUILDER_BIN"
}

teardown_file() {
  make clean
}

@test "[build] write procfile" {
  skip "This test does not run correctly in Github Actions due to use of embedded docker"
  run $LAMBDA_BUILDER_BIN build --working-directory tests/go --write-procfile
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ -f tests/go/Procfile ]]
}

@test "[build] dotnet6" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/dotnet6
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] go" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/go
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] go without go modules" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/go-nomod
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] hooks" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/hooks
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] lambda.yml" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/lambda.yml
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] lambda.yml-invalid-image" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/lambda.yml-invalid-image
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 1 ]]
}

@test "[build] lambda.yml-nonexistent-builder" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/lambda.yml-nonexistent-builder
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 1 ]]
}

@test "[build] npm" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/npm
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] nonexistent" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/nonexistent
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 1 ]]
}

@test "[build] non-detected" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/non-detected
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 1 ]]
}

@test "[build] pip" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/pip
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] pip-runtime" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/pip-runtime
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] pipenv" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/pipenv
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] poetry" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/poetry
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}

@test "[build] ruby" {
  run $LAMBDA_BUILDER_BIN build --working-directory tests/ruby
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
}
