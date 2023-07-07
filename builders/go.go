package builders

import "lambda-builder/io"

type GoBuilder struct {
	Config Config
}

func NewGoBuilder(config Config) (GoBuilder, error) {
	var err error
	config.BuilderBuildImage, err = getBuildImage(config, "lambci/lambda:build-go1.x")
	if err != nil {
		return GoBuilder{}, err
	}

	config.BuilderRunImage, err = getRunImage(config, "mlupin/docker-lambda:provided.al2")
	if err != nil {
		return GoBuilder{}, err
	}

	return GoBuilder{
		Config: config,
	}, nil
}

func (b GoBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "go.mod") {
		return true
	}

	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "main.go") {
		return true
	}

	return false
}

func (b GoBuilder) Execute() error {
	b.Config.HandlerMap = b.GetHandlerMap()
	return executeBuilder(b.script(), b.Config)
}

func (b GoBuilder) GetBuildImage() string {
	return b.Config.BuilderBuildImage
}

func (b GoBuilder) GetConfig() Config {
	return b.Config
}

func (b GoBuilder) GetHandlerMap() map[string]string {
	return map[string]string{
		"bootstrap": "bootstrap",
	}
}

func (b GoBuilder) Name() string {
	return "go"
}

func (b GoBuilder) script() string {
	return `
#!/usr/bin/env bash
set -eo pipefail

[ "$BUILDER_XTRACE" ] && set -o xtrace

indent() {
  sed -u "s/^/       /"
}

puts-step() {
  echo "-----> $*"
}

install-gomod() {
  if [[ -f "go.mod" ]]; then
    puts-step "Downloading dependencies via go mod"
    go mod download 2>&1 | indent
  else
    puts-step "Missing go.mod, downloading dependencies via go get"
    go get
  fi

  puts-step "Compiling via go build"
  go build -o bootstrap main.go 2>&1 | indent
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
  zip -q -r lambda.zip bootstrap
}

cp -a /var/task/. /go/src/handler
hook-pre-compile
install-gomod
hook-post-compile
hook-package
`
}
