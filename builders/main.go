package builders

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"lambda-builder/io"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"gopkg.in/yaml.v2"
)

type Builder interface {
	GetConfig() Config
	Detect() bool
	Execute() error
	BuildImage() string
	Name() string
}

type Config struct {
	BuildImage       string
	Identifier       string
	RunQuiet         bool
	WorkingDirectory string
}

type LambdaYML struct {
	Builder    string `yaml:"builder"`
	BuildImage string `yaml:"build_image"`
}

func executeBuilder(script string, config Config) error {
	args := []string{
		"container",
		"run",
		"--rm",
		"--env", "LAMBDA_BUILD_ZIP=1",
		"--label", "com.dokku.lambda-builder/executor=true",
		"--name", fmt.Sprintf("lambda-builder-executor-%s", config.Identifier),
		"--volume", fmt.Sprintf("%s:/tmp/task", config.WorkingDirectory),
		config.BuildImage,
		"/bin/bash", "-c", script,
	}

	cmd := execute.ExecTask{
		Args:        args,
		Command:     "docker",
		Cwd:         config.WorkingDirectory,
		StreamStdio: !config.RunQuiet,
	}

	res, err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute builder: %s", err.Error())
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("failed to execute builder, exit code %d", res.ExitCode)
	}

	return nil
}

func ParseLambdaYML(config Config) (LambdaYML, error) {
	var lambdaYML LambdaYML
	if !io.FileExistsInDirectory(config.WorkingDirectory, "lambda.yml") {
		return lambdaYML, nil
	}

	f, err := os.Open(filepath.Join(config.WorkingDirectory, "lambda.yml"))
	if err != nil {
		return lambdaYML, fmt.Errorf("error opening lambda.yml: %s", err.Error())
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return lambdaYML, fmt.Errorf("error reading lambda.yml: %s", err.Error())
	}

	if err := yaml.Unmarshal(bytes, &lambdaYML); err != nil {
		return lambdaYML, fmt.Errorf("error unmarshaling lambda.yml: %s", err.Error())
	}

	return lambdaYML, nil
}

func getBuilder(config Config, defaultImage string) (string, error) {
	if config.BuildImage != "" {
		return config.BuildImage, nil
	}

	lambdaYML, err := ParseLambdaYML(config)
	if err != nil {
		return "", err
	}

	return lambdaYML.BuildImage, nil
}
