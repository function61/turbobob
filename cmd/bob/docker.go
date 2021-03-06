package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/function61/turbobob/pkg/dockertag"
	"github.com/function61/turbobob/pkg/versioncontrol"
)

func isDevContainerRunning(containerName string) bool {
	result, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).CombinedOutput()
	if err != nil {
		// TODO: check for other errors
		return false
	}

	return strings.TrimRight(string(result), "\n") == "true"
}

func devContainerName(bobfile *Bobfile, builder BuilderSpec) string {
	return containerNameInternal("dev", bobfile, builder)
}

func langServerContainerName(bobfile *Bobfile, builder BuilderSpec) string {
	return containerNameInternal("langserver", bobfile, builder)
}

// do not use directly
func containerNameInternal(kind string, bobfile *Bobfile, builder BuilderSpec) string {
	return fmt.Sprintf("tb%s-%s-%s", kind, bobfile.ProjectName, builder.Name)

}

func builderImageName(bobfile *Bobfile, builder BuilderSpec) string {
	builderType, ref, err := parseBuilderUsesType(builder.Uses)
	if err != nil {
		panic(err)
	}

	switch builderType {
	case builderUsesTypeImage:
		return ref // "image:tag"
	case builderUsesTypeDockerfile:
		return "tb-" + bobfile.ProjectName + "-builder-" + builder.Name
	default:
		panic("unknown builderType")
	}
}

func buildBuilder(bobfile *Bobfile, builder *BuilderSpec) error {
	imageName := builderImageName(bobfile, *builder)

	builderUsesType, dockerfilePath, err := parseBuilderUsesType(builder.Uses)
	if err != nil {
		return err
	}

	if builderUsesType != builderUsesTypeDockerfile {
		return errors.New("buildBuilder(): incorrect uses type")
	}

	printHeading(fmt.Sprintf("Building builder %s (as %s)", builder.Name, imageName))

	var imageBuildCmd *exec.Cmd = nil

	// provide Dockerfile from stdin for contextless build
	if builder.ContextlessBuild {
		dockerfileContent, err := ioutil.ReadFile(dockerfilePath)
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
			"--file", dockerfilePath,
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
	revisionId *versioncontrol.RevisionId,
	publishArtefacts bool,
	builder BuilderSpec,
	envsAreRequired bool,
	osArches OsArchesSpec,
	fastbuild bool,
) ([]string, error) {
	env := func(key, value string) {
		dockerArgs = append(dockerArgs, "--env", key+"="+value)
	}

	env("FRIENDLY_REV_ID", revisionId.FriendlyRevisionId)
	env("REV_ID", revisionId.RevisionId)
	env("REV_ID_SHORT", revisionId.RevisionIdShort)

	for _, envKey := range builder.PassEnvs {
		envValue := os.Getenv(envKey)
		if envValue != "" {
			dockerArgs = append(dockerArgs, "--env", envKey)
		} else if envsAreRequired {
			return nil, envVarMissingErr(envKey)
		}
	}

	for envKey, envValue := range builder.Envs {
		env(envKey, envValue)
	}

	// BUILD_LINUX_AMD64=true, BUILD_LINUX_ARM=true, ...
	for _, buildEnv := range osArches.AsBuildEnvVariables() {
		env(buildEnv, "true")
	}

	if fastbuild {
		env("FASTBUILD", "true")
	}

	return dockerArgs, nil
}

func loginToDockerRegistry(dockerImage DockerImageSpec, cache *dockerRegistryLoginCache) error {
	credentialsObtainer := getDockerCredentialsObtainer(dockerImage)
	creds, err := credentialsObtainer.Obtain()
	if err != nil {
		return err
	}

	tagParsed := dockertag.Parse(dockerImage.Image)
	if tagParsed == nil {
		return ErrUnableToParseDockerTag
	}

	registryDefaulted := tagParsed.Registry
	if registryDefaulted == "" {
		registryDefaulted = dockertag.DockerHubHostname // docker.io
	}

	cacheKey := newDockerRegistryLoginCacheKey(registryDefaulted, *creds)

	if cache.Cached(cacheKey) {
		return nil
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

	cache.Cache(cacheKey)

	return nil
}

type dockerRegistryLoginCacheKey string

func newDockerRegistryLoginCacheKey(registry string, creds DockerCredentials) dockerRegistryLoginCacheKey {
	return dockerRegistryLoginCacheKey(fmt.Sprintf("%s:%s:%s", registry, creds.Username, creds.Password))
}

type dockerRegistryLoginCache struct {
	items map[dockerRegistryLoginCacheKey]bool
}

func newDockerRegistryLoginCache() *dockerRegistryLoginCache {
	return &dockerRegistryLoginCache{
		items: map[dockerRegistryLoginCacheKey]bool{},
	}
}

func (d *dockerRegistryLoginCache) Cached(key dockerRegistryLoginCacheKey) bool {
	_, cached := d.items[key]
	return cached
}

func (d *dockerRegistryLoginCache) Cache(key dockerRegistryLoginCacheKey) {
	d.items[key] = true
}
