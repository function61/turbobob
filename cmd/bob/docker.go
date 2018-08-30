package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func isDevContainerRunning(containerName string) bool {
	result, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).CombinedOutput()
	if err != nil {
		// TODO: check for other errors
		return false
	}

	return strings.TrimRight(string(result), "\n") == "true"
}

func devContainerName(bobfile *Bobfile, builderName string) string {
	return "tbdev-" + bobfile.ProjectName + "-" + builderName
}

func builderDockerfilePath(builderName string) string {
	return "Dockerfile." + builderName + "-build"
}

func builderImageName(bobfile *Bobfile, builderName string) string {
	return "tb-" + bobfile.ProjectName + "-builder-" + builderName
}

func buildBuilder(bobfile *Bobfile, builder *BuilderSpec) error {
	dockerfileContent, err := ioutil.ReadFile(builderDockerfilePath(builder.Name))
	if err != nil {
		return err
	}

	imageName := builderImageName(bobfile, builder.Name)

	// provide Dockerfile from stdin for contextless build
	imageBuildCmd := exec.Command("docker", "build", "-t", imageName, "-")
	imageBuildCmd.Stdin = bytes.NewBuffer(dockerfileContent)
	imageBuildCmd.Stdout = os.Stdout
	imageBuildCmd.Stderr = os.Stderr

	if err := imageBuildCmd.Run(); err != nil {
		return err
	}

	return nil
}

func dockerRelayEnvVars(dockerArgs []string, build *BuildMetadata, publishArtefacts bool, envs []string) ([]string, error) {
	dockerArgs = append(dockerArgs, "--env", "FRIENDLY_REV_ID="+build.FriendlyRevisionId)

	if publishArtefacts {
		dockerArgs = append(dockerArgs, "--env", "PUBLISH_ARTEFACTS=true")
	}

	for _, envKey := range envs {
		envValue := os.Getenv(envKey)
		if envValue == "" {
			return nil, envVarMissingErr(envKey)
		}

		dockerArgs = append(dockerArgs, "--env", envKey)
	}

	return dockerArgs, nil
}

var dockerCredsRe = regexp.MustCompile("^([^:]+):(.+)")

func loginToDockerHub() error {
	credsParts := dockerCredsRe.FindStringSubmatch(os.Getenv("DOCKER_CREDS"))
	if len(credsParts) != 3 {
		return ErrDockerCredsEnvNotSet
	}

	username := credsParts[1]
	password := credsParts[2]

	printHeading(fmt.Sprintf("Logging in to Docker Hub as %s", username))

	loginCmd := passthroughStdoutAndStderr(exec.Command(
		"docker",
		"login",
		"--username", username,
		"--password", password))

	if err := loginCmd.Run(); err != nil {
		return err
	}

	return nil
}
