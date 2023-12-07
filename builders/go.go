package builders

import "lambda-builder/io"

type GoBuilder struct {
	Config Config
}

func NewGoBuilder(config Config) (GoBuilder, error) {
	var err error
	defaultBuilder := "golang:1.21-bookworm"
	if !io.FileExistsInDirectory(config.WorkingDirectory, "go.mod") {
		defaultBuilder = "golang:1.17-bullseye"
	}

	config.BuilderBuildImage, err = getBuildImage(config, defaultBuilder)
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
  CGO_ENABLED=0 go build -o bootstrap main.go 2>&1 | indent
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

  if ! command -v zip >/dev/null 2>&1; then
    puts-step "Installing zip dependency for packaging"
    apt update && apt install -y --no-install-recommends zip
  fi

  puts-step "Creating package at lambda.zip"
  zip -q -r lambda.zip bootstrap
  mv lambda.zip /var/task/lambda.zip
}

cp -a /var/task/. /go/src/handler
cd /go/src/handler
hook-pre-compile
install-gomod
hook-post-compile
hook-package
`
}
