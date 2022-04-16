package builders

import "lambda-builder/io"

type RubyBuilder struct {
	Config Config
}

func NewRubyBuilder(config Config) RubyBuilder {
	if config.BuildImage == "" {
		config.BuildImage = "mlupin/docker-lambda:ruby2.7-build"
	}

	return RubyBuilder{
		Config: config,
	}
}

func (b RubyBuilder) BuildImage() string {
	return b.Config.BuildImage
}

func (b RubyBuilder) GetConfig() Config {
	return b.Config
}

func (b RubyBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "Gemfile.lock") {
		return true
	}

	return false
}

func (b RubyBuilder) Execute() error {
	return executeBuilder(b.script(), b.Config)
}

func (b RubyBuilder) Name() string {
	return "ruby"
}

func (b RubyBuilder) script() string {
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

install-bundler() {
  puts-step "Downloading dependencies via bundler"
  bundle config set --local path 'vendor/bundle' 2>&1 | indent
  bundle install 2>&1 | indent
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
install-bundler
hook-package
`
}
