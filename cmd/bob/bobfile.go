package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	bobfileName                = "turbobob.json"
	fileDescriptionBoilerplate = "https://github.com/function61/turbobob"
)

type Bobfile struct {
	FileDescriptionBoilerplate string            `json:"for_description_of_this_file_see"`
	VersionMajor               int               `json:"version_major"`
	ProjectName                string            `json:"project_name"`
	Builders                   []BuilderSpec     `json:"builders"`
	DockerImages               []DockerImageSpec `json:"docker_images"`
}

type BuilderSpec struct {
	Name             string   `json:"name"`
	DockerfilePath   string   `json:"dockerfile_path"`
	MountSource      string   `json:"mount_source"`
	MountDestination string   `json:"mount_destination"`
	DevCommand       []string `json:"dev_command"`
	DevPorts         []string `json:"dev_ports"`
	DevProTips       []string `json:"dev_pro_tips"`
	PassEnvs         []string `json:"pass_envs"`
	ContextlessBuild bool     `json:"contextless_build"`
}

type DockerImageSpec struct {
	Image          string `json:"image"`
	DockerfilePath string `json:"dockerfile_path"`
	AuthType       string `json:"auth_type"` // creds_from_env | aws_ecr
}

// FIXME: Bobfile should actually be read only after correct
// revision has been checked out from VCs
func readBobfile() (*Bobfile, error) {
	bobfileFile, err := os.Open(bobfileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBobfileNotFound
		}

		return nil, err
	}
	defer bobfileFile.Close()

	decoder := json.NewDecoder(bobfileFile)
	decoder.DisallowUnknownFields()

	bobfile := &Bobfile{}
	if err := decoder.Decode(bobfile); err != nil {
		return nil, err
	}

	if bobfile.VersionMajor != 1 {
		return nil, ErrUnsupportedBobfileVersion
	}

	if bobfile.FileDescriptionBoilerplate != fileDescriptionBoilerplate {
		return nil, ErrIncorrectFileDescriptionBp
	}

	if err := assertUniqueBuilderNames(bobfile); err != nil {
		return nil, err
	}

	return bobfile, nil
}

func assertUniqueBuilderNames(bobfile *Bobfile) error {
	alreadySeenNames := map[string]bool{}

	for _, builder := range bobfile.Builders {
		if _, alreadyExists := alreadySeenNames[builder.Name]; alreadyExists {
			return fmt.Errorf("duplicate builder name: %s", builder.Name)
		}

		alreadySeenNames[builder.Name] = true
	}

	return nil
}

func findBuilder(bobfile *Bobfile, builderName string) *BuilderSpec {
	for _, builder := range bobfile.Builders {
		if builder.Name == builderName {
			return &builder
		}
	}

	return nil
}
