package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/function61/gokit/builtin"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

func writeGitHubActionsBoilerplate() error {
	return writeBoilerplate(".github/workflows/build.yml", `name: Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2

    - name: Build
      run: |
        curl --fail --location --silent --output bob https://function61.com/go/turbobob-latest-linux-amd64 && chmod +x bob
        ./bob build in-ci-autodetect-settings
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
`)
}

func writeGitLabBoilerplate() error {
	fmt.Fprintln(os.Stderr, "WARN: in-ci-autodetect-settings not updated to support GitLab")

	return writeBoilerplate(".gitlab-ci.yml", `# Minimal Gitlab CI conf for Turbo Bob handoff
# For help with problems: https://github.com/function61/turbobob/blob/master/docs/ci_gitlab.md

build:
  script:
    - apk add --no-cache curl git
    - curl --fail --location --output bob https://function61.com/go/turbobob-latest-linux-amd64 && chmod +x bob
    - DOCKER_HOST="tcp://docker:2375" ./bob build in-ci-autodetect-settings
  image: docker:18.06-dind
  services:
    - docker:dind
  tags:
    - docker
`)
}

func writeBoilerplate(filePath string, content string) error {
	if dir := filepath.Dir(filePath); dir != "" { // make dir if it doesn't exist
		if err := os.MkdirAll(dir, osutil.FileMode(osutil.OwnerRWX, osutil.GroupRWX, osutil.OtherNone)); err != nil {
			return err
		}
	}

	exists, errExistsCheck := osutil.Exists(filePath)
	if errExistsCheck != nil {
		return errExistsCheck
	}

	if exists {
		return fmt.Errorf("CI boilerplate '%s' already exists", filePath)
	}

	return ioutil.WriteFile(
		filePath,
		[]byte(content),
		osutil.FileMode(osutil.OwnerRW, osutil.GroupRW, osutil.OtherNone))
}

func writeDefaultBobfile(producesDockerImage bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// guess project name from current workdir's basename
	projectName := filepath.Base(cwd)

	dockerImages := []DockerImageSpec{}
	if producesDockerImage {
		dockerImages = append(dockerImages, DockerImageSpec{
			Image:          "yourcompany/" + projectName,
			DockerfilePath: "Dockerfile",
		})
	}

	defaults := Bobfile{
		FileDescriptionBoilerplate: fileDescriptionBoilerplate,
		VersionMajor:               currentVersionMajor,
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
		DockerImages: dockerImages,
	}

	return writeBobfileIfNotExists(defaults)
}

func writeBobfileIfNotExists(content Bobfile) error {
	exists, errExistsCheck := osutil.Exists(bobfileName)
	if errExistsCheck != nil {
		return errExistsCheck
	}

	if exists {
		return ErrInitBobfileExists
	}

	asJson, errJson := json.MarshalIndent(&content, "", "\t")
	if errJson != nil {
		return errJson
	}

	return ioutil.WriteFile(
		bobfileName,
		[]byte(fmt.Sprintf("%s\n", asJson)),
		osutil.FileMode(osutil.OwnerRW, osutil.GroupRW, osutil.OtherNone))
}

func initEntry() *cobra.Command {
	return initEntryWithHidden(false)
}

func initEntryWithHidden(hidden bool) *cobra.Command {
	github := false
	gitLab := false
	docker := false

	cmd := &cobra.Command{
		Use:    "init",
		Short:  "Initializes this project with a default turbobob.json",
		Args:   cobra.NoArgs,
		Hidden: hidden,
		Run: func(cmd *cobra.Command, args []string) {
			if github {
				osutil.ExitIfError(writeGitHubActionsBoilerplate())
			}

			if gitLab {
				osutil.ExitIfError(writeGitLabBoilerplate())
			}

			osutil.ExitIfError(writeDefaultBobfile(docker))
		},
	}

	cmd.Flags().BoolVarP(&github, "github", "", github, "Write GitHub Actions boilerplate")
	cmd.Flags().BoolVarP(&gitLab, "gitlab", "", gitLab, "Write GitLab CI boilerplate")
	cmd.Flags().BoolVarP(&docker, "docker", "", docker, "This project should produce a Docker image?")

	cmd.AddCommand(&cobra.Command{
		Use:   "guess",
		Short: "Initialize turbobob.json by guessing configuration from Dockerfile",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(ErrorWrap("initGuessFromDockerfile", initGuessFromDockerfile()))
		},
	})

	return cmd
}
