package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/function61/gokit/app/dynversion"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

const (
	travisFilePath = ".travis.yml"
	gitlabFilePath = ".gitlab-ci.yml"
)

func writeTravisBoilerplate() error {
	exists, errExistsCheck := osutil.Exists(travisFilePath)
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
  - curl --fail --location --output bob https://function61.com/go/turbobob-latest-linux-amd64 && chmod +x bob
  - CI_REVISION_ID="$TRAVIS_COMMIT" ./bob build --publish-artefacts
`

	return ioutil.WriteFile(travisFilePath, []byte(boilerplate), 0600)
}

func writeGitLabBoilerplate() error {
	exists, errExistsCheck := osutil.Exists(gitlabFilePath)
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
    - curl --fail --location --output bob https://function61.com/go/turbobob-latest-linux-amd64 && chmod +x bob
    - CI_REVISION_ID="$CI_COMMIT_SHA" DOCKER_HOST="tcp://docker:2375" ./bob build --publish-artefacts
  image: docker:18.06-dind
  services:
    - docker:dind
  tags:
    - docker
`

	return ioutil.WriteFile(gitlabFilePath, []byte(boilerplate), 0600)
}

func writeDefaultBobfile(producesDockerImage bool) error {
	exists, errExistsCheck := osutil.Exists(bobfileName)
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
				osutil.ExitIfError(ErrInitingWithBobDevVersion)
			}

			if travis {
				osutil.ExitIfError(writeTravisBoilerplate())
			}

			if gitLab {
				osutil.ExitIfError(writeGitLabBoilerplate())
			}

			osutil.ExitIfError(writeDefaultBobfile(docker))
		},
	}

	cmd.Flags().BoolVarP(&travis, "travis", "", travis, "Write Travis CI boilerplate")
	cmd.Flags().BoolVarP(&gitLab, "gitlab", "", gitLab, "Write GitLab CI boilerplate")
	cmd.Flags().BoolVarP(&docker, "docker", "", docker, "This project should produce a Docker image?")
	cmd.Flags().BoolVarP(&ignoreDevWarning, "ignore-dev-warning", "", ignoreDevWarning, "Don't complain about initing with Bob's dev version")

	return cmd
}
