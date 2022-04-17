package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"lambda-builder/builders"
	"lambda-builder/io"
	"lambda-builder/ui"

	"github.com/google/uuid"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type BuildCommand struct {
	command.Meta

	quiet            bool
	workingDirectory string
}

func (c *BuildCommand) Name() string {
	return "build"
}

func (c *BuildCommand) Synopsis() string {
	return "Builds a lambda function"
}

func (c *BuildCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *BuildCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Builds a lambda.zip for the current directory": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *BuildCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	return args
}

func (c *BuildCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *BuildCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *BuildCommand) FlagSet() *flag.FlagSet {
	workingDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.quiet, "quiet", false, "run builder in quiet mode")
	f.StringVar(&c.workingDirectory, "working-directory", workingDirectory, "working directory")
	return f
}

func (c *BuildCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--count": complete.PredictNothing,
			"--quiet": complete.PredictNothing,
		},
	)
}

func (c *BuildCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	var err error
	c.workingDirectory, err = filepath.Abs(c.workingDirectory)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	logger, ok := c.Ui.(*ui.ZerologUi)
	if !ok {
		c.Ui.Error("Unable to fetch logger from cli")
		return 1
	}

	if !io.FolderExists(c.workingDirectory) {
		c.Ui.Error(fmt.Sprintf("Working directory '%s' does not exist", c.workingDirectory))
		return 1
	}

	if io.FileExistsInDirectory(c.workingDirectory, "lambda.zip") {
		c.Ui.Warn("Removing existing lambda.zip from working directory")
		os.Remove(filepath.Join(c.workingDirectory, "lambda.zip"))
	}

	identifier := uuid.New().String()
	config := builders.Config{
		Identifier:       identifier,
		RunQuiet:         c.quiet,
		WorkingDirectory: c.workingDirectory,
	}

	logger.LogHeader1("Detecting builder")
	builder, err := detectBuilder(config)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Detected %s builder", builder.Name()))

	logger.LogHeader1(fmt.Sprintf("Building app with image %s", builder.BuildImage()))
	if err := builder.Execute(); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	zipPath := filepath.Join(c.workingDirectory, "lambda.zip")
	logger.LogHeader1(fmt.Sprintf("Wrote %s", zipPath))
	sizeInBytes, err := io.FileSize(zipPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting filesize for %s: %s", zipPath, err.Error()))
		return 1
	}

	sizeInKB := io.BytesToKilobytes(sizeInBytes)
	sizeInMB := io.BytesToMegabytes(sizeInBytes)
	if sizeInMB >= 50 {
		c.Ui.Warn(fmt.Sprintf("Surpassed AWS Lambda 50MB zip file limit: %dMB (%dKB)", sizeInMB, sizeInKB))
		c.Ui.Warn("Consider using Docker Images for lambda function distribution")
	} else {
		c.Ui.Info(fmt.Sprintf("Current zip file size: %dMB (%dKB)", sizeInMB, sizeInKB))
	}

	return 0
}

func detectBuilder(config builders.Config) (builders.Builder, error) {
	var builder builders.Builder
	var err error
	bs := []builders.Builder{}

	lambdaYML, err := builders.ParseLambdaYML(config)
	if err != nil {
		return nil, err
	}

	builder, err = builders.NewDotnetBuilder(config)
	if err != nil {
		return nil, err
	}
	bs = append(bs, builder)

	builder, err = builders.NewGoBuilder(config)
	if err != nil {
		return nil, err
	}
	bs = append(bs, builder)

	builder, err = builders.NewNodejsBuilder(config)
	if err != nil {
		return nil, err
	}
	bs = append(bs, builder)

	builder, err = builders.NewPythonBuilder(config)
	if err != nil {
		return nil, err
	}
	bs = append(bs, builder)

	builder, err = builders.NewRubyBuilder(config)
	if err != nil {
		return nil, err
	}
	bs = append(bs, builder)

	for _, builder := range bs {
		if lambdaYML.Builder != "" && lambdaYML.Builder != builder.Name() {
			continue
		}

		if builder.Detect() {
			return builder, nil
		}
	}

	return nil, errors.New("no builder detected")
}
