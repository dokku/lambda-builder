# lambda-builder

A tool for building lamda functions into uploadable zip files via Docker based on work from [@lambci](https://github.com/lambci/docker-lambda) and [@mLupine](https://github.com/mLupine/docker-lambda).

## Why?

I don't want to go through the motions of figuring out the correct way to build my app for AWS. I suspect there are others out there who feel the same.

## Dependencies

- The `docker` binary
- Golang 1.7+

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0
```

## Usage

```
Usage: lambda-builder [--version] [--help] <command> [<args>]

Available commands are:
    build      Builds a lambda function
    version    Return the version of the binary
```

To build an app:

```shell
cd path/to/app

# will write a lambda.zip in the current working directory
lambda-builder build
```

Alternatively, a given path can be specified via the `--working-directory` flag:

```shell
# will write a lambda.zip in the specified path
lambda-builder build --working-directory path/to/app
```

In addition to the `lambda.yml`, a docker image can be produced from the generated artifact by specifying the `--build-image` flag. This also allows for multiple `--label`  flags as well as specifying a single image tag via either `-t` or `--tag`:

```shell
# will write a lambda.zip in the specified path
# and generate a docker image named `lambda-builder:$APP:latest`
# where $APP is the last portion of the working directory
lambda-builder build --build-image

# adds the labels com.example/key=value and com.example/another-key=value
lambda-builder build --build-image --label com.example/key=value --label com.example/another-key=value

# tags the image as app/awesome:1234
lambda-builder build --build-image --tag app/awesome:1234
```

### How does it work

Internally, `lambda-builder` detects a given language and builds the app according to the script specified by the detected builder within a disposablecontainer environment emulating AWS Lambda. If a builder is not detected, the build will fail. The following languages are supported:

- `dotnet`
  - default build image: `mlupin/docker-lambda:dotnet6-build`
  - requirement: `Function.cs`
  - runtimes:
    - dotnet6
    - dotnetcore3.1
- `go`
  - default build image: `lambci/lambda:build-go1.x`
  - requirement: `go.mod`
  - runtimes:
    - provided.al2
- `nodejs`
  - default build image: `mlupin/docker-lambda:nodejs14.x-build`
  - requirement: `package-lock.json`
  - runtimes:
    - nodejs12.x
    - nodejs14.x
- `python`
  - default build image: `mlupin/docker-lambda:python3.9-build`
  - requirement: `requirements.txt`, `poetry.lock`, or `Pipfile.lock`
  - notes: Autodetects the python version from `poetry.lock`, `Pipfile.lock`, or `runtime.txt`
  - runtimes:
    - python3.8
    - python3.9
- `ruby`
  - default build image: `mlupin/docker-lambda:ruby2.7-build`
  - requirement: `Gemfile.lock`
  - runtimes:
    - ruby2.7

When the app is built, a `lambda.zip` will be produced in the specified working directory. The resulting `lambda.zip` can be uploaded to S3 and used within a Lambda function.

Both the builder and the build image environment can be overriden in an optional `lambda.yml` file in the specified working directory.

### `lambda.yml`

The following a short description of the `lambda.yml` format.

```yaml
---
build_image: mlupin/docker-lambda:dotnetcore3.1-build
builder: dotnet
run_image: mlupin/docker-lambda:dotnetcore3.1
```

- `build_image`: A docker image that is accessible by the docker daemon. The `build_image` _should_ be based on an existing Lambda image - builders may fail if they cannot run within the specified `build_image`. The build will fail if the image is inaccessible by the docker daemon.
- `builder`: The name of a builder. This may be used if multiple builders match and a specific builder is desired. If an invalid builder is specified, the build will fail.
- `run_image`: A docker image that is accessible by the docker daemon. The `run_image` _should_ be based on an existing Lambda image - built images may fail to start if they are not compatible with the produced artifact. The generation of the `run` iage will fail if the image is inaccessible by the docker daemon.

### Deploying

The `lambda.zip` file can be directly uploaded to a lambda function and used as is by specifying the correct runtime. See the `test.bats` files in any of the `test` examples for more info on how to perform this with the `awscli` (v2).

## Examples

See the `tests` directory for examples on how to use this project.
