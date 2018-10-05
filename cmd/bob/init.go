package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	travisFilePath = ".travis.yml"
	gitlabFilePath = ".gitlab-ci.yml"
)

func writeTravisBoilerplate() error {
	exists, errExistsCheck := fileExists(travisFilePath)
	if errExistsCheck != nil {
		return errExistsCheck
	}

	if exists {
		return ErrCiFileAlreadyExists
	}

	boilerplate := `# Minimal Travis conf to bootstrap Turbo Bob

sudo: required
services: docker
language: minimal
script:
  - curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/_VERSION_/bob_linux-amd64 && chmod +x bob
  - CI_REVISION_ID="$TRAVIS_COMMIT" ./bob build --publish-artefacts
`

	boilerplateReplaced := strings.Replace(boilerplate, "_VERSION_", version, -1)

	return ioutil.WriteFile(travisFilePath, []byte(boilerplateReplaced), 0600)
}

func writeGitLabBoilerplate() error {
	exists, errExistsCheck := fileExists(gitlabFilePath)
	if errExistsCheck != nil {
		return errExistsCheck
	}

	if exists {
		return ErrCiFileAlreadyExists
	}

	// https://gitlab.com/ayufan/container-registry/blob/master/.gitlab-ci.yml
	// https://gitlab.com/gitlab-org/gitlab-runner/issues/1250
	// https://stackoverflow.com/questions/39608736/docker-in-docker-with-gitlab-shared-runner-for-building-and-pushing-docker-image
	boilerplate := `# Minimal Gitlab CI conf to bootstrap Turbo Bob

build:
  script:
    - apk add --no-cache curl git
    - curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/20180919_1430_dae525b2/bob_linux-amd64 && chmod +x bob
    - CI_REVISION_ID="$CI_COMMIT_SHA" DOCKER_HOST="tcp://docker:2375" ./bob build --publish-artefacts
  image: docker:18.06-dind
  services:
    - docker:dind
  tags:
    - docker
`

	boilerplateReplaced := strings.Replace(boilerplate, "_VERSION_", version, -1)

	return ioutil.WriteFile(gitlabFilePath, []byte(boilerplateReplaced), 0600)
}

func writeDefaultBobfile(producesDockerImage bool) error {
	exists, errExistsCheck := fileExists(bobfileName)
	if errExistsCheck != nil {
		return errExistsCheck
	}

	if exists {
		return ErrInitBobfileExists
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// guess project name from current workdir's basename
	projectName := filepath.Base(cwd)

	defaults := Bobfile{
		FileDescriptionBoilerplate: "https://github.com/function61/turbobob",
		VersionMajor:               1,
		ProjectName:                projectName,
		Builders: []BuilderSpec{
			{
				Name:             "default",
				MountDestination: "/app",
				PassEnvs:         []string{},
				DevCommand:       []string{"bash"},
				DevProTips:       []string{},
				DevPorts:         []string{},
			},
		},
		DockerImages: []DockerImageSpec{},
	}

	if producesDockerImage {
		defaults.DockerImages = append(defaults.DockerImages, DockerImageSpec{
			Image:          "yourcompany/" + projectName,
			DockerfilePath: "Dockerfile",
			AuthType:       "creds_from_env",
		})
	}

	asJson, errJson := json.MarshalIndent(&defaults, "", "\t")
	if errJson != nil {
		return errJson
	}

	return ioutil.WriteFile(
		bobfileName,
		[]byte(fmt.Sprintf("%s\n", asJson)),
		0700)
}

func initEntry() *cobra.Command {
	travis := false
	gitLab := false
	docker := false

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes this project with a default turbobob.json",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if travis {
				reactToError(writeTravisBoilerplate())
			}

			if gitLab {
				reactToError(writeGitLabBoilerplate())
			}

			reactToError(writeDefaultBobfile(docker))
		},
	}

	cmd.Flags().BoolVarP(&travis, "travis", "", travis, "Write Travis CI boilerplate")
	cmd.Flags().BoolVarP(&gitLab, "gitlab", "", gitLab, "Write GitLab CI boilerplate")
	cmd.Flags().BoolVarP(&docker, "docker", "", docker, "This project should produce a Docker image?")

	return cmd
}
