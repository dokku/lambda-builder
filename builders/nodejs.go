package builders

import "lambda-builder/io"

type NodejsBuilder struct {
	Config Config
}

func NewNodejsBuilder(config Config) (NodejsBuilder, error) {
	var err error
	config.BuilderBuildImage, err = getBuildImage(config, "mlupin/docker-lambda:nodejs14.x-build")
	if err != nil {
		return NodejsBuilder{}, err
	}

	config.BuilderRunImage, err = getRunImage(config, "mlupin/docker-lambda:nodejs14.x")
	if err != nil {
		return NodejsBuilder{}, err
	}

	return NodejsBuilder{
		Config: config,
	}, nil
}

func (b NodejsBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "package-lock.json") {
		return true
	}

	return false
}

func (b NodejsBuilder) Execute() error {
	b.Config.HandlerMap = b.GetHandlerMap()
	return executeBuilder(b.script(), b.GetTaskBuildDir(), b.Config)
}

func (b NodejsBuilder) GetBuildImage() string {
	return b.Config.BuilderBuildImage
}

func (b NodejsBuilder) GetConfig() Config {
	return b.Config
}

func (b NodejsBuilder) GetHandlerMap() map[string]string {
	return map[string]string{
		"function.js":        "function.handler",
		"lambda_function.js": "lambda_function.handler",
	}
}

func (b NodejsBuilder) GetTaskBuildDir() string {
	return "/var/task"
}

func (b NodejsBuilder) Name() string {
	return "nodejs"
}

func (b NodejsBuilder) script() string {
	return `
#!/usr/bin/env bash
set -eo pipefail

indent() {
  sed -u "s/^/       /"
}

puts-header() {
  echo "=====> $*"
}

puts-step() {
  echo "-----> $*"
}

install-npm() {
  puts-step "Installing dependencies via npm"
  npm install 2>&1 | indent
}

hook-package() {
  if [[ "$LAMBDA_BUILD_ZIP" != "1" ]]; then
    return
  fi

  puts-step "Creating package at lambda.zip"
  zip -q -r lambda.zip ./*
  mv lambda.zip /tmp/task/lambda.zip
  rm -rf lambda.zip
}

cp -a /tmp/task/. /var/task
install-npm
hook-package
`
}
