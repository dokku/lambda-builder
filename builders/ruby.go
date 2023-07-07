package builders

import "lambda-builder/io"

type RubyBuilder struct {
	Config Config
}

func NewRubyBuilder(config Config) (RubyBuilder, error) {
	var err error
	config.BuilderBuildImage, err = getBuildImage(config, "mlupin/docker-lambda:ruby2.7-build")
	if err != nil {
		return RubyBuilder{}, err
	}

	config.BuilderRunImage, err = getRunImage(config, "mlupin/docker-lambda:ruby2.7")
	if err != nil {
		return RubyBuilder{}, err
	}

	return RubyBuilder{
		Config: config,
	}, nil
}

func (b RubyBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "Gemfile.lock") {
		return true
	}

	return false
}

func (b RubyBuilder) GetBuildImage() string {
	return b.Config.BuilderBuildImage
}

func (b RubyBuilder) GetConfig() Config {
	return b.Config
}

func (b RubyBuilder) GetHandlerMap() map[string]string {
	return map[string]string{
		"function.rb":        "function.handler",
		"lambda_function.rb": "lambda_function.handler",
	}
}

func (b RubyBuilder) Execute() error {
	b.Config.HandlerMap = b.GetHandlerMap()
	return executeBuilder(b.script(), b.Config)
}

func (b RubyBuilder) Name() string {
	return "ruby"
}

func (b RubyBuilder) script() string {
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

install-bundler() {
  puts-step "Downloading dependencies via bundler"
  bundle config set --local path 'vendor/bundle' 2>&1 | indent
  bundle install 2>&1 | indent
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
install-bundler
hook-post-compile
hook-package
`
}
