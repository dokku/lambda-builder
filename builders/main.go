package builders

import (
	"fmt"

	execute "github.com/alexellis/go-execute/pkg/v1"
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
	Export           bool
	Identifier       string
	RunImage         string
	RunQuiet         bool
	WorkingDirectory string
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
