package builders

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"lambda-builder/io"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
)

type PythonBuilder struct {
	Config Config
}

func NewPythonBuilder(config Config) PythonBuilder {
	if config.BuildImage == "" {
		version, err := parsePythonVersion(config.WorkingDirectory, []string{"3.8", "3.9"})
		if err != nil {
			panic(err)
		}

		config.BuildImage = fmt.Sprintf("mlupin/docker-lambda:python%s-build", version)
	}

	if config.RunImage == "" {
		version, err := parsePythonVersion(config.WorkingDirectory, []string{"3.8", "3.9"})
		if err != nil {
			panic(err)
		}

		config.RunImage = fmt.Sprintf("mlupin/docker-lambda:python%s", version)
	}

	return PythonBuilder{
		Config: config,
	}
}

func (b PythonBuilder) BuildImage() string {
	return b.Config.BuildImage
}

func (b PythonBuilder) GetConfig() Config {
	return b.Config
}

func (b PythonBuilder) Detect() bool {
	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "requirements.txt") {
		return true
	}

	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "poetry.lock") {
		return true
	}

	if io.FileExistsInDirectory(b.Config.WorkingDirectory, "Pipfile.lock") {
		return true
	}

	return false
}

func (b PythonBuilder) Execute() error {
	return executeBuilder(b.script(), b.Config)
}

func (b PythonBuilder) Name() string {
	return "python"
}

func (b PythonBuilder) script() string {
	return `
#!/usr/bin/env bash
set -eo pipefail

# Tell Python to not buffer it's stdin/stdout.
export PYTHONUNBUFFERED=1

indent() {
  sed -u "s/^/       /"
}

puts-header() {
  echo "=====> $*"
}

puts-step() {
  echo "-----> $*"
}

puts-warning() {
  echo " !     $*"
}

install-pip() {
  puts-step "Installing dependencies via pip"
  version="$(python-major-minor)"
  mkdir -p ".venv/lib/python${version}"
  pip install --target ".venv/lib/python${version}/site-packages" -r requirements.txt 2>&1 | indent
}

install-pipenv() {
  puts-step "Creating virtualenv"
  virtualenv -p python .venv | indent

  puts-step "Installing dependencies via pipenv"
  export PIPENV_VENV_IN_PROJECT=1
  version="$(python-major-minor)"

  if [[ ! -f "Pipfile.lock" ]]; then
    pipenv install --skip-lock 2>&1 | indent
  else
    pipenv install --deploy 2>&1 | indent
  fi
}

install-poetry() {
  puts-step "Installing dependencies via poetry"
  poetry config virtualenvs.create true
  poetry config virtualenvs.in-project true
  poetry install --no-dev 2>&1 | indent
}

python-major-minor() {
  python -c 'import sys; print(str(sys.version_info[0])+"."+str(sys.version_info[1]))'
}

cleanup-deps() {
  puts-step "Writing dependencies to correct path"
  version="$(python-major-minor)"
  find "/var/task/.venv/lib/python${version}/site-packages" -type f -print0 | xargs -0 chmod 644
  find "/var/task/.venv/lib/python${version}/site-packages" -type d -print0 | xargs -0 chmod 755
  pushd "/var/task/.venv/lib/python${version}/site-packages" >/dev/null || return 1
  cp -a --no-clobber "/var/task/.venv/lib/python${version}/site-packages/." /var/task
  popd >/dev/null || return 1
  rm -rf /var/task/.venv
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

if [[ -f "requirements.txt" ]]; then
  install-pip
elif [[ -f "Pipfile" ]]; then
  install-pipenv
elif [[ -f "poetry.lock" ]] || [[ -f "pyproject.toml" ]]; then
  install-poetry
else
	puts-warning "No dependency file detected"
	exit 1
fi

cleanup-deps
hook-package
`
}

func parsePythonVersion(workingDirectory string, supportedPythonVersions []string) (string, error) {
	var err error
	version := "3.9"
	if io.FileExistsInDirectory(workingDirectory, "Pipfile.lock") {
		version, err = parsePythonVersionFromPipfileLock(workingDirectory)
	}

	if io.FileExistsInDirectory(workingDirectory, "poetry.lock") {
		version, err = parsePythonVersionFromPoetryLock(workingDirectory, supportedPythonVersions)
	}

	if io.FileExistsInDirectory(workingDirectory, "runtime.txt") {
		version, err = parsePythonVersionFromRuntimeTxt(workingDirectory)
	}

	if err != nil {
		return "", err
	}

	constraint, err := semver.NewConstraint(version)
	if err != nil {
		return "", fmt.Errorf("error parsing version python constraint: %s", err.Error())
	}

	for _, version := range supportedPythonVersions {
		v, err := semver.NewVersion(version)
		if err != nil {
			return "", fmt.Errorf("error parsing supported python version '%s': %s", version, err.Error())
		}

		if constraint.Check(v) {
			return fmt.Sprintf("%d.%d", v.Major(), v.Minor()), nil
		}
	}

	return version, err
}

func parsePythonVersionFromPipfileLock(workingDirectory string) (string, error) {
	f, err := os.Open(filepath.Join(workingDirectory, "Pipfile.lock"))
	if err != nil {
		return "", fmt.Errorf("error opening Pipefile.lock: %s", err.Error())
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("error reading Pipefile.lock: %s", err.Error())
	}

	type PipefileLock struct {
		Meta struct {
			Requires struct {
				PythonVersion string `json:"python_version"`
			} `json:"requires"`
		} `json:"_meta"`
	}
	var pipefileLock PipefileLock
	if err := json.Unmarshal(bytes, &pipefileLock); err != nil {
		return "", err
	}

	if pipefileLock.Meta.Requires.PythonVersion == "" {
		return "3.9", nil
	}

	return pipefileLock.Meta.Requires.PythonVersion, nil
}

func parsePythonVersionFromPoetryLock(workingDirectory string, supportedPythonVersions []string) (string, error) {
	f, err := os.Open(filepath.Join(workingDirectory, "poetry.lock"))
	if err != nil {
		return "", fmt.Errorf("error opening poetry.lock: %s", err.Error())
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("error reading poetry.lock: %s", err.Error())
	}

	type PoetryLock struct {
		Metadata struct {
			PythonVersions string `toml:"python-versions"`
		} `toml:"metadata"`
	}
	var poetryLock PoetryLock
	if err := toml.Unmarshal(bytes, &poetryLock); err != nil {
		return "", fmt.Errorf("error unmarshaling poetry.lock: %s", err.Error())
	}

	if poetryLock.Metadata.PythonVersions == "" || poetryLock.Metadata.PythonVersions == "*" {
		return "3.9", nil
	}

	if v, err := semver.NewVersion(poetryLock.Metadata.PythonVersions); err == nil {
		poetryLock.Metadata.PythonVersions = fmt.Sprintf("%d.%d", v.Major(), v.Minor())
	}

	constraint, err := semver.NewConstraint(poetryLock.Metadata.PythonVersions)
	if err != nil {
		return "", fmt.Errorf("error parsing poetry.lock python constraint: %s", err.Error())
	}

	for _, version := range supportedPythonVersions {
		v, err := semver.NewVersion(version)
		if err != nil {
			return "", fmt.Errorf("error parsing supported python version '%s': %s", version, err.Error())
		}

		if constraint.Check(v) {
			return version, nil
		}
	}

	return "", fmt.Errorf("no valid python version found")
}

func parsePythonVersionFromRuntimeTxt(workingDirectory string) (string, error) {
	f, err := os.Open(filepath.Join(workingDirectory, "runtime.txt"))
	if err != nil {
		return "", fmt.Errorf("error opening runtime.txt: %s", err.Error())
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("error reading runtime.txt: %s", err.Error())
	}

	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	if len(lines) != 1 {
		return "", fmt.Errorf("error parsing runtime.txt, expected 1 line, found %d", len(lines))
	}

	parts := strings.SplitN(lines[0], "-", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("error parsing runtime.txt, contents: %v", lines[0])
	}

	v, err := semver.NewVersion(parts[1])
	if err != nil {
		return "", fmt.Errorf("error parsing semver from runtime.txt: %s", err.Error())
	}

	return fmt.Sprintf("%d.%d", v.Major(), v.Minor()), nil
}
