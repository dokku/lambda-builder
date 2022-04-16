package builders

import "lambda-builder/io"

type DotnetBuilder struct {
	Config Config
}

func NewDotnetBuilder(config Config) (DotnetBuilder, error) {
	var err error
	config.BuildImage, err = getBuilder(config, "mlupin/docker-lambda:dotnet6-build")
	if err != nil {
		return DotnetBuilder{}, err
	}

	return DotnetBuilder{
		Config: config,
	}, nil
}

func (b DotnetBuilder) BuildImage() string {
	return b.Config.BuildImage
}

func (b DotnetBuilder) GetConfig() Config {
	return b.Config
}

func (b DotnetBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "Function.cs") {
		return true
	}

	return false
}

func (b DotnetBuilder) Execute() error {
	return executeBuilder(b.script(), b.Config)
}

func (b DotnetBuilder) Name() string {
	return "dotnet"
}

func (b DotnetBuilder) script() string {
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

install-dotnet() {
  puts-step "Compiling via dotnet publish"
  dotnet publish -c Release -o pub 2>&1 | indent
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
install-dotnet
hook-package
`
}
