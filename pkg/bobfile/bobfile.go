// Project configuration file for Turbo Bob
package bobfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	. "github.com/function61/gokit/builtin"
	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/turbobob/pkg/versioncontrol"
)

const (
	Name                       = ".config/turbobob.json"
	FileDescriptionBoilerplate = "https://github.com/function61/turbobob"
)

const (
	CurrentVersionMajor = 1
)

type Bobfile struct {
	FileDescriptionBoilerplate string            `json:"for_description_of_this_file_see"`
	VersionMajor               int               `json:"version_major"`
	ProjectName                string            `json:"project_name"`
	Builders                   []BuilderSpec     `json:"builders"`
	DockerImages               []DockerImageSpec `json:"docker_images,omitempty"`
	Subrepos                   []SubrepoSpec     `json:"subrepos,omitempty"`
	OsArches                   *OsArchesSpec     `json:"os_arches,omitempty"`
	Experiments                experiments       `json:"experiments_i_consent_to_breakage,omitempty"`
	Meta                       ProjectMetadata   `json:"meta,omitempty"`
	Deprecated1                string            `json:"project_emoji_icon,omitempty"` // moved to `ProjectMetadata`
}

func (b Bobfile) ProjectEmojiIcon() string {
	return firstNonEmpty(b.Meta.ProjectEmojiIcon, b.Deprecated1)
}

type ProjectMetadata struct {
	Description      string       `json:"description,omitempty"`        // what this project is used for
	Website          string       `json:"website,omitempty"`            // URL of homepage or such
	Documentation    string       `json:"documentation,omitempty"`      // URL of documentation website
	ProjectEmojiIcon string       `json:"project_emoji_icon,omitempty"` // to quickly differentiate projects in e.g. workspace switcher
	License          *LicenseInfo `json:"license,omitempty"`            // which license this project is licensed under
}

// when experiments are removed or graduated to production, they will be removed from here
// (yielding unknown field error) and breaking the build. the price of opting in to unstable stuff.
type experiments struct {
	PrepareStep bool `json:"prepare_step,omitempty"`
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
	LinuxRiscv64   bool `json:"linux-riscv64"`
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
	maybeAppend(o.LinuxRiscv64, "BUILD_LINUX_RISCV64")

	maybeAppend(o.DarwinAmd64, "BUILD_DARWIN_AMD64")

	return ret
}

/*
		Suppose you have three builders: 1) backend, 2) frontend and 3) documentation.
		Here's the order in which the commands are executed:


	    Start ────────────────┐     ┌───────────────┐     ┌───────────────┐
	                          │     ▲               │     ▲               │
	               ┌──────────▼┐    │    ┌──────────▼┐    │    ┌──────────▼┐
	       Backend │  Prepare ││    │    │  Build   ││    │    │  Publish ││
	               └──────────┼┘    │    └──────────┼┘    │    └──────────┼┘
	                          │     │               │     │               │
	                          │     │               │     │               │
	               ┌──────────▼┐    │    ┌──────────▼┐    │    ┌──────────▼┐
	      Frontend │  Prepare ││    │    │  Build   ││    │    │  Publish ││
	               └──────────┼┘    │    └──────────┼┘    │    └──────────┼┘
	                          │     │               │     │               │
	                          │     │               │     │               │
	               ┌──────────▼┐    │    ┌──────────▼┐    │    ┌──────────▼┐
	 Documentation │  Prepare ││    │    │  Build   ││    │    │  Publish ││
	               └──────────┼┘    │    └──────────┼┘    │    └──────────┼┘
	                          │     │               │     │               │
	                          │     │               │     │               │
	                          └─────┘               └─────┘               ▼

		Rationale:

		- backend needs some codegenerated stuff from documentation, like URLs so backend can link to documentation,
		  so backend build can use stuff from documentation.prepare step.
		- you'll want to publish artefacts only if all builders succeeded (*.build before *.publish),
		  so there's no unnecessary uploads.
*/
type BuilderCommands struct {
	Prepare []string `json:"prepare,omitempty"`
	Build   []string `json:"build"`
	Publish []string `json:"publish,omitempty"`
	Dev     []string `json:"dev"`
}

type BuilderSpec struct {
	Name             string            `json:"name"`
	Uses             string            `json:"uses"` // "docker://alpine:latest" | "dockerfile://build-default.Dockerfile"
	MountSource      string            `json:"mount_source,omitempty"`
	MountDestination string            `json:"mount_destination"`
	Workdir          string            `json:"workdir,omitempty"`
	Commands         BuilderCommands   `json:"commands"`
	DevPorts         []string          `json:"dev_ports,omitempty"`
	DevHttpIngress   string            `json:"dev_http_ingress,omitempty"`
	DevProTips       []string          `json:"dev_pro_tips,omitempty"`
	DevShellCommands []DevShellCommand `json:"dev_shell_commands,omitempty"` // injected as history for quick recall (ctrl + r)
	Envs             map[string]string `json:"env,omitempty"`
	PassEnvs         []string          `json:"pass_envs,omitempty"`
	ContextlessBuild bool              `json:"contextless_build,omitempty"`
}

type DevShellCommand struct {
	Command   string `json:"command"`
	Important bool   `json:"important"` // important commands are shown as pro-tips on "$ bob dev"
}

type DockerImageSpec struct {
	Image          string   `json:"image"`
	DockerfilePath string   `json:"dockerfile_path"`
	AuthType       *string  `json:"auth_type"`           // creds_from_env
	Platforms      []string `json:"platforms,omitempty"` // if set, uses buildx
	TagLatest      bool     `json:"tag_latest"`
}

type LicenseInfo struct {
	Expression string               `json:"expression"` // https://spdx.github.io/spdx-spec/v2-draft/SPDX-license-expressions/
	Rationale  LicenseInfoRationale `json:"rationale"`
}

type LicenseInfoRationale struct {
	Kind                 LicenseInfoRationaleKind                  `json:"kind"`
	Notes                string                                    `json:"notes,omitempty"`
	AutodetectedFromFile *LicenseInfoRationaleAutodetectedFromFile `json:"autodetected_from_file,omitempty"`
}

type LicenseInfoRationaleAutodetectedFromFile struct {
	File       string  `json:"file"`   // "LICENSE"
	Digest     string  `json:"digest"` // "sha256:..."
	Confidence float64 `json:"confidence,omitempty"`
}

type LicenseInfoRationaleKind string

const (
	LicenseInfoRationaleKindAutodetectedFromFile LicenseInfoRationaleKind = "autodetected_from_file"
	LicenseInfoRationaleKindDefinedManually      LicenseInfoRationaleKind = "defined_manually"
)

// FIXME: Bobfile should actually be read only after correct
// revision has been checked out from VCs
func Read() (*Bobfile, error) {
	return ReadWithAllowLegacyOption(true)
}

// same as `Read`, but takes in option to whether to allow reading the Bobfile from the legacy (root) dir
func ReadWithAllowLegacyOption(allowLegacyFilename bool) (*Bobfile, error) {
	withErr := func(err error) (*Bobfile, error) { return nil, fmt.Errorf("bobfile.Read: %w", err) }

	bobfileFile, err := func() (io.ReadCloser, error) {
		bobfileFile, err := os.Open(Name)
		if err != nil && errors.Is(err, fs.ErrNotExist) && allowLegacyFilename { // try old filename
			return os.Open("turbobob.json")
		}

		return bobfileFile, err
	}()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return withErr(ErrBobfileNotFound)
		} else {
			return withErr(err)
		}
	}
	defer bobfileFile.Close()

	bobfile := &Bobfile{}
	if err := jsonfile.UnmarshalDisallowUnknownFields(bobfileFile, bobfile); err != nil {
		return withErr(err)
	}

	if bobfile.VersionMajor != CurrentVersionMajor {
		return withErr(ErrUnsupportedBobfileVersion)
	}

	if bobfile.FileDescriptionBoilerplate != FileDescriptionBoilerplate {
		return withErr(ErrIncorrectFileDescriptionBp)
	}

	if err := validateBuilders(bobfile); err != nil {
		return withErr(ErrorWrap("validateBuilders", err))
	}

	for _, subrepo := range bobfile.Subrepos {
		// https://stackoverflow.com/questions/19633763/unmarshaling-json-in-golang-required-field
		// we cannot even check for empty value in custom type's UnmarshalJSON() because if
		// the value is missing, the func does not get called. IOW unmarshaling can still
		// end up in broken data, so we must check it manually..
		if err := ErrorIfUnset(subrepo.Kind == "", "subrepo.Kind"); err != nil {
			return withErr(err)
		}
	}

	if bobfile.OsArches == nil {
		bobfile.OsArches = &OsArchesSpec{}
	}

	return bobfile, nil
}

func validateBuilders(bobfile *Bobfile) error {
	alreadySeenNames := map[string]Void{}

	for _, builder := range bobfile.Builders {
		if _, alreadyExists := alreadySeenNames[builder.Name]; alreadyExists {
			return fmt.Errorf("duplicate builder name: %s", builder.Name)
		}

		alreadySeenNames[builder.Name] = Void{}

		if len(builder.Commands.Prepare) > 0 && !bobfile.Experiments.PrepareStep {
			return fmt.Errorf("%s: you need to opt-in to prepare_step experiment", builder.Name)
		}
	}

	return nil
}
