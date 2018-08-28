package main

import (
	"encoding/json"
	"os"
)

type BuilderSpec struct {
	Name             string   `json:"name"`
	MountDestination string   `json:"mount_destination"`
	DevCommand       []string `json:"dev_command"`
	PassEnvs         []string `json:"pass_envs"`
}

type Bobfile struct {
	VersionMajor int           `json:"version_major"`
	ProjectName  string        `json:"project_name"`
	Builders     []BuilderSpec `json:"builders"`
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
	bobfileFile, err := os.Open("bob.json")
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
