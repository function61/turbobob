package main

import (
	"fmt"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/turbobob/pkg/versioncontrol"
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
	Subrepos                   []SubrepoSpec     `json:"subrepos"`
	OsArches                   *OsArchesSpec     `json:"os_arches"`
}

type SubrepoSpec struct {
	Source      string              `json:"source"`
	Kind        versioncontrol.Kind `json:"kind"`
	Destination string              `json:"destination"`
	Revision    string              `json:"revision"`
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
	DarwinAmd64    bool `json:"darwin-amd64"`
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

	maybeAppend(o.DarwinAmd64, "BUILD_DARWIN_AMD64")

	return ret
}

type BuilderCommands struct {
	Build   []string `json:"build"`
	Publish []string `json:"publish"`
	Dev     []string `json:"dev"`
}

type BuilderSpec struct {
	Name             string            `json:"name"`
	Uses             string            `json:"uses"` // "docker://alpine:latest" | "dockerfile://build-default.Dockerfile"
	MountSource      string            `json:"mount_source"`
	MountDestination string            `json:"mount_destination"`
	Workdir          string            `json:"workdir"`
	Commands         BuilderCommands   `json:"commands"`
	DevPorts         []string          `json:"dev_ports"`
	DevProTips       []string          `json:"dev_pro_tips"`
	Envs             map[string]string `json:"env"`
	PassEnvs         []string          `json:"pass_envs"`
	ContextlessBuild bool              `json:"contextless_build"`
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

	bobfile := &Bobfile{}
	if err := jsonfile.Unmarshal(bobfileFile, bobfile, true); err != nil {
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

	for _, subrepo := range bobfile.Subrepos {
		// https://stackoverflow.com/questions/19633763/unmarshaling-json-in-golang-required-field
		// we cannot even check for empty value in custom type's UnmarshalJSON() because if
		// the value is missing, the func does not get called. IOW unmarshaling can still
		// end up in broken data, so we must check it manually..
		if subrepo.Kind == "" {
			return nil, fmt.Errorf("Subrepo Kind cannot be empty")
		}
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
