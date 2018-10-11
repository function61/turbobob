package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

func builderDockerfilePath(builder *BuilderSpec) string {
	return "Dockerfile." + builder.Name + "-build"
}

func builderImageName(bobfile *Bobfile, builderName string) string {
	return "tb-" + bobfile.ProjectName + "-builder-" + builderName
}

func buildBuilder(bobfile *Bobfile, builder *BuilderSpec) error {
	imageName := builderImageName(bobfile, builder.Name)

	var imageBuildCmd *exec.Cmd = nil

	// provide Dockerfile from stdin for contextless build
	if builder.ContextlessBuild {
		dockerfileContent, err := ioutil.ReadFile(builderDockerfilePath(builder))
		if err != nil {
			return err
		}

		// FIXME: would "--file -" be more semantic?
		imageBuildCmd = exec.Command(
			"docker",
			"build",
			"--tag", imageName,
			"-")
		imageBuildCmd.Stdin = bytes.NewBuffer(dockerfileContent)
		imageBuildCmd.Stdout = os.Stdout
		imageBuildCmd.Stderr = os.Stderr
	} else {
		imageBuildCmd = exec.Command(
			"docker",
			"build",
			"--tag", imageName,
			"--file", builderDockerfilePath(builder),
			".")
		imageBuildCmd.Stdout = os.Stdout
		imageBuildCmd.Stderr = os.Stderr
	}

	if err := imageBuildCmd.Run(); err != nil {
		return err
	}

	return nil
}

func dockerRelayEnvVars(
	dockerArgs []string,
	build *BuildMetadata,
	publishArtefacts bool,
	envs []string,
	envsAreRequired bool,
) ([]string, error) {
	dockerArgs = append(dockerArgs, "--env", "FRIENDLY_REV_ID="+build.FriendlyRevisionId)

	if publishArtefacts {
		dockerArgs = append(dockerArgs, "--env", "PUBLISH_ARTEFACTS=true")
	}

	for _, envKey := range envs {
		envValue := os.Getenv(envKey)
		if envValue != "" {
			dockerArgs = append(dockerArgs, "--env", envKey)
		} else if envsAreRequired {
			return nil, envVarMissingErr(envKey)
		}
	}

	return dockerArgs, nil
}

func loginToDockerRegistry(dockerImage DockerImageSpec) error {
	credentialsObtainer := getDockerCredentialsObtainer(dockerImage)
	creds, err := credentialsObtainer.Obtain()
	if err != nil {
		return err
	}

	tagParsed := ParseDockerTag(dockerImage.Image)
	if tagParsed == nil {
		return ErrUnableToParseDockerTag
	}

	registryDefaulted := tagParsed.Registry
	if registryDefaulted == "" {
		registryDefaulted = "docker.io"
	}

	printHeading(fmt.Sprintf("Logging in as %s to %s", creds.Username, registryDefaulted))

	loginCmd := passthroughStdoutAndStderr(exec.Command(
		"docker",
		"login",
		"--username", creds.Username,
		"--password", creds.Password,
		registryDefaulted))

	if err := loginCmd.Run(); err != nil {
		return err
	}

	return nil
}
