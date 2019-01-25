package main

import (
	"encoding/json"
	"fmt"
	"github.com/function61/gokit/dynversion"
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

	boilerplate := `# Minimal Travis conf for Turbo Bob handoff
# For help with problems: https://github.com/function61/turbobob/blob/master/docs/ci_travis.md

sudo: required
services: docker
language: minimal
script:
  - curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/_VERSION_/bob_linux-amd64 && chmod +x bob
  - CI_REVISION_ID="$TRAVIS_COMMIT" ./bob build --publish-artefacts
`

	boilerplateReplaced := strings.Replace(boilerplate, "_VERSION_", dynversion.Version, -1)

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

	boilerplate := `# Minimal Gitlab CI conf for Turbo Bob handoff
# For help with problems: https://github.com/function61/turbobob/blob/master/docs/ci_gitlab.md

build:
  script:
    - apk add --no-cache curl git
    - curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/_VERSION_/bob_linux-amd64 && chmod +x bob
    - CI_REVISION_ID="$CI_COMMIT_SHA" DOCKER_HOST="tcp://docker:2375" ./bob build --publish-artefacts
  image: docker:18.06-dind
  services:
    - docker:dind
  tags:
    - docker
`

	boilerplateReplaced := strings.Replace(boilerplate, "_VERSION_", dynversion.Version, -1)

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
		FileDescriptionBoilerplate: fileDescriptionBoilerplate,
		VersionMajor:               1,
		ProjectName:                projectName,
		Builders: []BuilderSpec{
			{
				Name:             "default",
				Uses:             "dockerfile://build-default.Dockerfile",
				MountSource:      "",
				MountDestination: "/app",
				PassEnvs:         []string{},
				Commands: BuilderCommands{
					Dev: []string{"bash"},
				},
				DevProTips:       []string{},
				DevPorts:         []string{},
				ContextlessBuild: false,
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
	ignoreDevWarning := false

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes this project with a default turbobob.json",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if dynversion.IsDevVersion() && !ignoreDevWarning {
				reactToError(ErrInitingWithBobDevVersion)
			}

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
	cmd.Flags().BoolVarP(&ignoreDevWarning, "ignore-dev-warning", "", ignoreDevWarning, "Don't complain about initing with Bob's dev version")

	return cmd
}
