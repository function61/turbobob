package main

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func writeTravisBoilerplate() error {
	boilerplate := `# Minimal Travis conf to bootstrap Turbo Bob

sudo: required
services: docker
language: minimal
script:
  - curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/_VERSION_/bob_linux-amd64
  - chmod +x bob
  - CI_REVISION_ID="$TRAVIS_COMMIT" ./bob build --publish-artefacts
`

	boilerplateReplaced := strings.Replace(boilerplate, "_VERSION_", version, -1)

	return ioutil.WriteFile(".travis.yml", []byte(boilerplateReplaced), 0600)
}

func writeDefaultBobfile() error {
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
				Name:     "default",
				PassEnvs: []string{},
			},
		},
	}

	asJson, errJson := json.MarshalIndent(&defaults, "", "\t")
	if errJson != nil {
		return errJson
	}

	return ioutil.WriteFile(bobfileName, asJson, 0700)
}

func initEntry() *cobra.Command {
	travis := false

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes this project with a default turbobob.json",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if travis {
				reactToError(writeTravisBoilerplate())
			}

			reactToError(writeDefaultBobfile())
		},
	}

	cmd.Flags().BoolVarP(&travis, "travis", "", travis, "Write Travis CI boilerplate")

	return cmd
}
