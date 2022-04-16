package builders

import "lambda-builder/io"

type NodejsBuilder struct {
	Config Config
}

func NewNodejsBuilder(config Config) NodejsBuilder {
	if config.BuildImage == "" {
		config.BuildImage = "mlupin/docker-lambda:nodejs14.x-build"
	}

	return NodejsBuilder{
		Config: config,
	}
}

func (b NodejsBuilder) BuildImage() string {
	return b.Config.BuildImage
}

func (b NodejsBuilder) GetConfig() Config {
	return b.Config
}

func (b NodejsBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "package-lock.json") {
		return true
	}
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "yarn.lock") {
		return true
	}

	return false
}

func (b NodejsBuilder) Execute() error {
	return executeBuilder(b.script(), b.Config)
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
}

cp -a /tmp/task/. /var/task
install-npm
hook-package
`
}
