package main

import (
	"encoding/json"
	"os"
)

const bobfileName = "turbobob.json"

type Bobfile struct {
	FileDescriptionBoilerplate string        `json:"for_description_of_this_file_see"`
	VersionMajor               int           `json:"version_major"`
	ProjectName                string        `json:"project_name"`
	Builders                   []BuilderSpec `json:"builders"`
}

type BuilderSpec struct {
	Name             string   `json:"name"`
	MountDestination string   `json:"mount_destination"`
	DevCommand       []string `json:"dev_command"`
	PassEnvs         []string `json:"pass_envs"`
}

func (b *BuilderSpec) MountDestinationOrDefaultToApp() string {
	if b.MountDestination != "" {
		return b.MountDestination
	}

	return "/app"
}

func (b *BuilderSpec) DevCommandOrDefaultToBash() []string {
	if len(b.DevCommand) == 0 {
		return []string{"bash"}
	}

	return b.DevCommand
}

func readBobfile() (*Bobfile, error) {
	bobfileFile, err := os.Open(bobfileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBobfileNotFound
		}

		return nil, err
	}
	defer bobfileFile.Close()

	var bobfile Bobfile
	if err := json.NewDecoder(bobfileFile).Decode(&bobfile); err != nil {
		return nil, err
	}

	if bobfile.VersionMajor != 1 {
		return nil, ErrUnsupportedBobfileVersion
	}

	return &bobfile, nil
}

func findBuilder(bobfile *Bobfile, builderName string) *BuilderSpec {
	for _, builder := range bobfile.Builders {
		if builder.Name == builderName {
			return &builder
		}
	}

	return nil
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		// unknown error. maybe error accessing FS?
		return false, err
	}

	return true, nil
}
