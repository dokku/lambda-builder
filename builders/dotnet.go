package builders

import "lambda-builder/io"

type DotnetBuilder struct {
	Config Config
}

func NewDotnetBuilder(config Config) (DotnetBuilder, error) {
	var err error
	config.BuilderBuildImage, err = getBuildImage(config, "mlupin/docker-lambda:dotnet6-build")
	if err != nil {
		return DotnetBuilder{}, err
	}

	config.BuilderRunImage, err = getRunImage(config, "mlupin/docker-lambda:dotnet6")
	if err != nil {
		return DotnetBuilder{}, err
	}

	return DotnetBuilder{
		Config: config,
	}, nil
}

func (b DotnetBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "Function.cs") {
		return true
	}

	return false
}

func (b DotnetBuilder) Execute() error {
	b.Config.HandlerMap = b.GetHandlerMap()
	return executeBuilder(b.script(), b.GetTaskBuildDir(), b.Config)
}

func (b DotnetBuilder) GetBuildImage() string {
	return b.Config.BuilderBuildImage
}

func (b DotnetBuilder) GetConfig() Config {
	return b.Config
}

func (b DotnetBuilder) GetHandlerMap() map[string]string {
	return map[string]string{}
}

func (b DotnetBuilder) GetTaskBuildDir() string {
	return "/var/task"
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

hook-pre-compile() {
  if [[ ! -f bin/pre_compile ]]; then
    return
  fi

  puts-step "Running pre-compile hook"
  chmod +x bin/pre_compile
  bin/pre_compile
}

hook-post-compile() {
  if [[ ! -f bin/post_compile ]]; then
    return
  fi

  puts-step "Running post-compile hook"
  chmod +x bin/post_compile
  bin/post_compile
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
hook-pre-compile
install-dotnet
hook-post-compile
hook-package
`
}
