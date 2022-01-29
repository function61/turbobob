package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	. "github.com/function61/gokit/builtin"
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

	imageBuildCmd, err := func() (*exec.Cmd, error) {
		// provide Dockerfile from stdin for contextless build
		if builder.ContextlessBuild {
			dockerfileContent, err := ioutil.ReadFile(dockerfilePath)
			if err != nil {
				return nil, err
			}

			// FIXME: would "--file -" be more semantic?
			cmd := exec.Command(
				"docker",
				"build",
				"--tag", imageName,
				"-")
			cmd.Stdin = bytes.NewBuffer(dockerfileContent)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			return cmd, nil
		} else {
			cmd := exec.Command(
				"docker",
				"build",
				"--tag", imageName,
				"--file", dockerfilePath,
				".")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			return cmd, nil
		}
	}()
	if err != nil {
		return err
	}

	if err := imageBuildCmd.Run(); err != nil {
		return err
	}

	return nil
}

func dockerRelayEnvVars(
	dockerArgs []string,
	revisionId *versioncontrol.RevisionId,
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

	if creds == nil {
		return nil
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

	//nolint:gosec // ok
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
	items map[dockerRegistryLoginCacheKey]Void
}

func newDockerRegistryLoginCache() *dockerRegistryLoginCache {
	return &dockerRegistryLoginCache{
		items: map[dockerRegistryLoginCacheKey]Void{},
	}
}

func (d *dockerRegistryLoginCache) Cached(key dockerRegistryLoginCacheKey) bool {
	_, cached := d.items[key]
	return cached
}

func (d *dockerRegistryLoginCache) Cache(key dockerRegistryLoginCacheKey) {
	d.items[key] = Void{}
}
