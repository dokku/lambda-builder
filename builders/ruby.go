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

func (b RubyBuilder) BuildImage() string {
	return b.Config.BuilderBuildImage
}

func (b RubyBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "Gemfile.lock") {
		return true
	}

	return false
}

func (b RubyBuilder) GetConfig() Config {
	return b.Config
}

func (b RubyBuilder) GetTaskBuildDir() string {
	return "/var/task"
}

func (b RubyBuilder) Execute() error {
	return executeBuilder(b.script(), b.GetTaskBuildDir(), b.Config)
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
  rm -rf lambda.zip
}

cp -a /tmp/task/. /var/task
install-bundler
hook-package
`
}
