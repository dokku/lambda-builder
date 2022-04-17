package builders

import "lambda-builder/io"

type GoBuilder struct {
	Config Config
}

func NewGoBuilder(config Config) (GoBuilder, error) {
	var err error
	config.BuildImage, err = getBuilder(config, "lambci/lambda:build-go1.x")
	if err != nil {
		return GoBuilder{}, err
	}

	return GoBuilder{
		Config: config,
	}, nil
}

func (b GoBuilder) BuildImage() string {
	return b.Config.BuildImage
}

func (b GoBuilder) GetConfig() Config {
	return b.Config
}

func (b GoBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "go.sum") {
		return true
	}

	return false
}

func (b GoBuilder) Execute() error {
	return executeBuilder(b.script(), b.Config)
}

func (b GoBuilder) Name() string {
	return "go"
}

func (b GoBuilder) script() string {
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

install-gomod() {
  puts-step "Downloading dependencies via go mod"
  go mod download 2>&1 | indent

  puts-step "Compiling via go build"
  go build -o bootstrap main.go 2>&1 | indent
}

hook-package() {
  if [[ "$LAMBDA_BUILD_ZIP" != "1" ]]; then
    return
  fi

  puts-step "Creating package at lambda.zip"
  zip -q -r lambda.zip bootstrap
  mv lambda.zip /tmp/task/lambda.zip
  rm -rf lambda.zip
}

cp -a /tmp/task/. /go/src/handler
install-gomod
hook-package
`
}
