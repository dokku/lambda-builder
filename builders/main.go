package builders

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lambda-builder/io"

	execute "github.com/alexellis/go-execute/pkg/v1"
	extract "github.com/codeclysm/extract/v3"
	"gopkg.in/yaml.v2"
)

type Builder interface {
	Detect() bool
	Execute() error
	GetBuildImage() string
	GetConfig() Config
	GetHandlerMap() map[string]string
	Name() string
}

type Config struct {
	BuildEnv          []string
	Builder           string
	BuilderBuildImage string
	BuilderRunImage   string
	GenerateRunImage  bool
	Handler           string
	HandlerMap        map[string]string
	Identifier        string
	ImageEnv          []string
	ImageLabels       []string
	ImageTag          string
	Port              int
	RunQuiet          bool
	WorkingDirectory  string
	WriteProcfile     bool
}

func (c Config) GetImageTag() string {
	if c.ImageTag != "" {
		return c.ImageTag
	}

	appName := filepath.Base(c.WorkingDirectory)
	return fmt.Sprintf("lambda-builder/%s:latest", appName)
}

type LambdaYML struct {
	Builder    string `yaml:"builder"`
	BuildImage string `yaml:"build_image"`
	RunImage   string `yaml:"run_image"`
}

func executeBuilder(script string, config Config) error {
	if err := executeBuildContainer(script, config); err != nil {
		return err
	}

	taskHostBuildDir, err := os.MkdirTemp("", "lambda-builder")
	if err != nil {
		return fmt.Errorf("error creating build dir: %w", err)
	}

	defer func() {
		os.RemoveAll(taskHostBuildDir)
	}()

	fmt.Printf("-----> Extracting lambda.zip into build context dir\n")
	zipPath := filepath.Join(config.WorkingDirectory, "lambda.zip")
	data, _ := ioutil.ReadFile(zipPath)
	buffer := bytes.NewBuffer(data)
	if err := extract.Zip(context.Background(), buffer, taskHostBuildDir, nil); err != nil {
		return fmt.Errorf("error extracting lambda.zip into build context dir: %w", err)
	}

	handler := getFunctionHandler(taskHostBuildDir, config)
	if config.WriteProcfile && !io.FileExistsInDirectory(taskHostBuildDir, "Procfile") {
		if handler == "" {
			fmt.Printf(" !     Unable to detect handler in build directory\n")
		} else {
			fmt.Printf("=====> Writing Procfile from handler: %s\n", handler)

			fmt.Printf("       Writing to working directory\n")
			if err := writeProcfile(handler, config.WorkingDirectory); err != nil {
				return fmt.Errorf("error writing Procfile to working directory: %w", err)
			}

			fmt.Printf("       Writing to build directory\n")
			if err := writeProcfile(handler, taskHostBuildDir); err != nil {
				return fmt.Errorf("error writing Procfile to temporary build directory: %w", err)
			}
		}
	}

	if config.GenerateRunImage {
		fmt.Printf("=====> Building image\n")
		fmt.Printf("       Generating temporary Dockerfile\n")

		dockerfilePath, err := ioutil.TempFile("", "lambda-builder")
		defer func() {
			os.Remove(dockerfilePath.Name())
		}()

		if err != nil {
			return fmt.Errorf("error generating temporary Dockerfile: %w", err)
		}

		if err := generateRunDockerfile(handler, config, dockerfilePath); err != nil {
			return err
		}

		fmt.Printf("       Executing build of %s\n", config.GetImageTag())
		if err := buildDockerImage(taskHostBuildDir, config, "run", dockerfilePath); err != nil {
			return err
		}
	}

	return nil
}

func executeBuildContainer(script string, config Config) error {
	fmt.Printf("       Generating temporary build script\n")
	scriptPath, err := os.Create(filepath.Join(config.WorkingDirectory, ".lambda-builder"))
	defer func() {
		os.Remove(scriptPath.Name())
	}()

	if err != nil {
		return fmt.Errorf("error generating temporary build script: %w", err)
	}

	if _, err := scriptPath.WriteString(strings.TrimSpace(script)); err != nil {
		return err
	}

	fmt.Printf("       Generating temporary Dockerfile\n")
	dockerfilePath, err := ioutil.TempFile("", "lambda-builder")
	defer func() {
		os.Remove(dockerfilePath.Name())
	}()

	if err != nil {
		return fmt.Errorf("error generating temporary Dockerfile: %w", err)
	}

	if err := generateBuildDockerfile(config, dockerfilePath, scriptPath); err != nil {
		return err
	}

	fmt.Printf("       Executing build of %s\n", config.GetImageTag())
	if err := buildDockerImage(config.WorkingDirectory, config, "build", dockerfilePath); err != nil {
		return err
	}

	defer func() {
		buildImageTag := fmt.Sprintf("%s-build", config.GetImageTag())
		fmt.Printf("       Removing build image: %s", buildImageTag)
		args := []string{
			"image",
			"rm",
			"--force",
			buildImageTag,
		}
		cmd := execute.ExecTask{
			Args:        args,
			Command:     "docker",
			Cwd:         config.WorkingDirectory,
			StreamStdio: !config.RunQuiet,
		}

		if _, err := cmd.Execute(); err != nil {
			fmt.Printf("       Error cleaning up build image: %s", err.Error())
		}
	}()

	extractLambdaFromBuildImage(config)

	return nil
}

func generateBuildDockerfile(config Config, dockerfilePath *os.File, scriptPath *os.File) error {
	tpl, err := template.New("t1").Parse(`
FROM {{ .build_image }}
LABEL com.dokku.lambda-builder/builder={{ .builder_name }}
ENV LAMBDA_BUILD_ZIP=1
WORKDIR /var/task
COPY . /var/task
{{range .env}}
ENV {{.}}
{{end}}
RUN mv {{ .build_script_name }} /usr/local/bin/build-lambda && \
	chmod +x /usr/local/bin/build-lambda && \
	head -n1 /usr/local/bin/build-lambda && \
	/usr/local/bin/build-lambda
`)
	if err != nil {
		return fmt.Errorf("error generating template: %s", err)
	}

	data := map[string]interface{}{
		"build_script_name": filepath.Base(scriptPath.Name()),
		"env":               config.BuildEnv,
		"builder":           config.Builder,
		"build_image":       config.BuilderBuildImage,
	}

	if err := tpl.Execute(dockerfilePath, data); err != nil {
		return fmt.Errorf("error writing Dockerfile: %s", err)
	}

	return nil
}

func extractLambdaFromBuildImage(config Config) error {
	args := []string{
		"container",
		"run",
		"--rm",
		"--label", "com.dokku.lambda-builder/extractor=true",
		"--name", fmt.Sprintf("lambda-builder-extractor-%s", config.Identifier),
		"--volume", fmt.Sprintf("%s:/tmp/task", config.WorkingDirectory),
	}

	args = append(args, fmt.Sprintf("%s-build", config.GetImageTag()), "/bin/bash", "-c", "mv /var/task/lambda.zip /tmp/task/lambda.zip")

	cmd := execute.ExecTask{
		Args:        args,
		Command:     "docker",
		Cwd:         config.WorkingDirectory,
		StreamStdio: !config.RunQuiet,
	}

	res, err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("error executing builder: %w", err)
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("error executing builder, exit code %d", res.ExitCode)
	}

	return nil
}

func generateRunDockerfile(cmd string, config Config, dockerfilePath *os.File) error {
	tpl, err := template.New("t1").Parse(`
FROM {{ .run_image }}
{{ if ne .port "-1" }}
ENV DOCKER_LAMBDA_API_PORT={{ .port }}
ENV DOCKER_LAMBDA_RUNTIME_PORT={{ .port }}
{{ end }}
{{range .env}}
ENV {{.}}
{{end}}
{{ if ne .command "" }}
CMD ["{{ .cmd }}"]
{{ end }}
COPY . /var/task
`)
	if err != nil {
		return fmt.Errorf("error generating template: %s", err)
	}

	data := map[string]interface{}{
		"cmd":       cmd,
		"env":       config.ImageEnv,
		"port":      strconv.Itoa(config.Port),
		"run_image": config.BuilderRunImage,
	}

	if err := tpl.Execute(dockerfilePath, data); err != nil {
		return fmt.Errorf("error writing Dockerfile: %s", err)
	}

	return nil
}

func buildDockerImage(directory string, config Config, phase string, dockerfilePath *os.File) error {
	imageTag := config.GetImageTag()
	if phase == "build" {
		imageTag = fmt.Sprintf("%s-build", config.GetImageTag())
	}

	args := []string{
		"image",
		"build",
		"--file", dockerfilePath.Name(),
		"--progress", "plain",
		"--tag", imageTag,
	}

	if phase == "run" {
		for _, label := range config.ImageLabels {
			args = append(args, "--label", label)
		}
	}

	args = append(args, directory)

	cmd := execute.ExecTask{
		Args:        args,
		Command:     "docker",
		Cwd:         config.WorkingDirectory,
		StreamStdio: !config.RunQuiet,
	}

	res, err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("error building image: %w", err)
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("error building image, exit code %d", res.ExitCode)
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
		return lambdaYML, fmt.Errorf("error opening lambda.yml: %w", err)
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return lambdaYML, fmt.Errorf("error reading lambda.yml: %w", err)
	}

	if err := yaml.Unmarshal(bytes, &lambdaYML); err != nil {
		return lambdaYML, fmt.Errorf("error unmarshaling lambda.yml: %w", err)
	}

	return lambdaYML, nil
}

func getBuildImage(config Config, defaultImage string) (string, error) {
	if config.BuilderBuildImage != "" {
		return config.BuilderBuildImage, nil
	}

	lambdaYML, err := ParseLambdaYML(config)
	if err != nil {
		return "", err
	}

	if lambdaYML.BuildImage == "" {
		return defaultImage, nil
	}

	return lambdaYML.BuildImage, nil
}

func getRunImage(config Config, defaultImage string) (string, error) {
	if config.BuilderRunImage != "" {
		return config.BuilderRunImage, nil
	}

	lambdaYML, err := ParseLambdaYML(config)
	if err != nil {
		return "", err
	}

	if lambdaYML.RunImage == "" {
		return defaultImage, nil
	}

	return lambdaYML.RunImage, nil
}
