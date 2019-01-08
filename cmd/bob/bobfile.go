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
	OsArches                   *OsArchesSpec     `json:"os_arches"`
}

// documents os/arch combos which this project's build artefacts support.
// follow constants from Go's GOOS/GOARCH
type OsArchesSpec struct {
	Neutral        bool `json:"neutral"`       // works across all OSes and arches, example: native JavaScript project or a website
	LinuxNeutral   bool `json:"linux-neutral"` // works on Linux, arch doesn't matter
	LinuxAmd64     bool `json:"linux-amd64"`
	LinuxArm       bool `json:"linux-arm"`
	LinuxArm64     bool `json:"linux-arm64"`
	WindowsNeutral bool `json:"windows-neutral"` // works on Windows, arch doesn't matter
	WindowsAmd64   bool `json:"windows-amd64"`
}

func (o *OsArchesSpec) AsBuildEnvVariables() []string {
	ret := []string{}

	maybeAppend := func(enabled bool, key string) {
		if enabled {
			ret = append(ret, key)
		}
	}

	maybeAppend(o.Neutral, "BUILD_NEUTRAL")

	maybeAppend(o.WindowsNeutral, "BUILD_WINDOWS_NEUTRAL")
	maybeAppend(o.WindowsAmd64, "BUILD_WINDOWS_AMD64")

	maybeAppend(o.LinuxNeutral, "BUILD_LINUX_NEUTRAL")
	maybeAppend(o.LinuxAmd64, "BUILD_LINUX_AMD64")
	maybeAppend(o.LinuxArm, "BUILD_LINUX_ARM")
	maybeAppend(o.LinuxArm64, "BUILD_LINUX_ARM64")

	return ret
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

	if bobfile.OsArches == nil {
		bobfile.OsArches = &OsArchesSpec{}
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
