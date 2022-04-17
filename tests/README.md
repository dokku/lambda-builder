# Test Apps

This directory contains test apps for use with lambda-builder.

Where possible, there is a `test.bats` file within the test app. This file will:

- build the lambda (using `lambda-builder` on the `$PATH`)
- create the iam role and attach the managed `AWSLambdaBasicExecutionRole` role
- create a lambda function with the proper runtime (as expected by lambda-builder) with the aforementioned iam role
- invoke the lambda function with a minimal payload
- except for Cloudwatch Log Groups, cleanup after itself when complete

Please be aware that all resources created will have the prefix `lambda-` and be tagged with the following tags:

- `app=lambda-builder`
- `com.dokku.lambda-builder/runtime=$LAMBDA_RUNTIME`
