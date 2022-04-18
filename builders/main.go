package builders

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"lambda-builder/io"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"gopkg.in/yaml.v2"
)

type Builder interface {
	Detect() bool
	Execute() error
	GetBuildImage() string
	GetConfig() Config
	GetHandlerMap() map[string]string
	GetTaskBuildDir() string
	Name() string
}

type Config struct {
	BuildEnv          []string
	Builder           string
	BuilderBuildImage string
	BuilderRunImage   string
	GenerateImage     bool
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

func executeBuilder(script string, taskBuildDir string, config Config) error {
	tmp, err := os.MkdirTemp("", "lambda-builder")
	defer func() {
		os.RemoveAll(tmp)
	}()

	if err != nil {
		return fmt.Errorf("error preparing temporary build directory: %s", err.Error())
	}

	if err := executeBuildContainer(tmp, script, taskBuildDir, config); err != nil {
		return err
	}

	handler := getFunctionHandler(tmp, config)
	if config.WriteProcfile && !io.FileExistsInDirectory(tmp, "Procfile") {
		if handler == "" {
			fmt.Printf(" !     Unable to detect handler in build directory\n")
		} else {
			fmt.Printf("=====> Writing Procfile from handler: %s\n", handler)

			fmt.Printf("       Writing to working directory\n")
			if err := writeProcfile(handler, config.WorkingDirectory); err != nil {
				return fmt.Errorf("error writing Procfile to working directory: %s", err.Error())
			}

			fmt.Printf("       Writing to build directory\n")
			if err := writeProcfile(handler, tmp); err != nil {
				return fmt.Errorf("error writing Procfile to temporary build directory: %s", err.Error())
			}
		}
	}

	if config.GenerateImage {
		fmt.Printf("=====> Building image\n")
		fmt.Printf("       Generating temporary Dockerfile\n")
		if err := generateDockerfile(handler, tmp, config); err != nil {
			return err
		}

		fmt.Printf("       Executing build of %s\n", config.GetImageTag())
		if err := buildDockerImage(tmp, config); err != nil {
			return err
		}
	}

	return nil
}

func executeBuildContainer(tmp string, script string, taskBuildDir string, config Config) error {
	args := []string{
		"container",
		"run",
		"--rm",
		"--env", "LAMBDA_BUILD_ZIP=1",
		"--label", "com.dokku.lambda-builder/executor=true",
		"--name", fmt.Sprintf("lambda-builder-executor-%s", config.Identifier),
		"--volume", fmt.Sprintf("%s:/tmp/task", config.WorkingDirectory),
		"--volume", fmt.Sprintf("%s:%s", tmp, taskBuildDir),
	}

	for _, envPair := range config.BuildEnv {
		args = append(args, "--env", envPair)
	}
	args = append(args, config.BuilderBuildImage, "/bin/bash", "-c", script)

	cmd := execute.ExecTask{
		Args:        args,
		Command:     "docker",
		Cwd:         config.WorkingDirectory,
		StreamStdio: !config.RunQuiet,
	}

	res, err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("error executing builder: %s", err.Error())
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("error executing builder, exit code %d", res.ExitCode)
	}

	return nil
}

func generateDockerfile(cmd string, directory string, config Config) error {
	dockerfileName := filepath.Join(directory, fmt.Sprintf("%s.Dockerfile", config.Identifier))
	f, err := os.Create(dockerfileName)
	if err != nil {
		return fmt.Errorf("error creating Dockerfile: %s", err)
	}

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

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("error writing Dockerfile: %s", err)
	}

	return nil
}

func buildDockerImage(directory string, config Config) error {
	args := []string{
		"image",
		"build",
		"--file", filepath.Join(directory, fmt.Sprintf("%s.Dockerfile", config.Identifier)),
		"--progress", "plain",
		"--tag", config.GetImageTag(),
	}

	for _, label := range config.ImageLabels {
		args = append(args, "--label", label)
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
		return fmt.Errorf("error building image: %s", err.Error())
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
