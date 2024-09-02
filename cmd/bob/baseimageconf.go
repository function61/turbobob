package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/bobfile"
)

const (
	baseImageJsonLocation = "/turbobob-baseimage.json"
)

// base image conf JSON - able to provide hints for useful commands, setting up cache paths etc.
type BaseImageConfig struct {
	DevShellCommands []bobfile.DevShellCommand `json:"dev_shell_commands"`
	PathsToCache     []string                  `json:"paths_to_cache"` // will be set up as symlinks to a persistent mountpoint, so that subsequent containers benefit from cache
	Langserver       *LangserverSpec           `json:"langserver"`

	FileDescriptionBoilerplate string `json:"for_description_of_this_file_see"` // URL to Bob homepage

	Deprecated1 []string `json:"langserver_cmd"` // DEPRECATED
}

type LangserverSpec struct {
	Command   []string `json:"command"`   // a command run inside the container to launch the language server
	Languages []string `json:"languages"` // e.g. "go". any of "source.<language>" ("source.go" => use "go") from https://github.com/github/linguist/blob/master/grammars.yml
}

// base image conf is optional. if it doesn't exist, an empty (but valid) conf will be
// returned without error
func loadBaseImageConfWhenInsideContainer() (*BaseImageConfig, error) {
	exists, err := osutil.Exists(baseImageJsonLocation)
	if err != nil {
		return nil, fmt.Errorf("loadBaseImageConfWhenInsideContainer: %w", err)
	}

	if !exists {
		return &BaseImageConfig{}, nil
	}

	conf := &BaseImageConfig{}
	if err := jsonfile.ReadDisallowUnknownFields(baseImageJsonLocation, conf); err != nil {
		return nil, fmt.Errorf("loadBaseImageConfWhenInsideContainer: %w", err)
	}

	baseImageConfMigrate(conf) // mutates

	return conf, nil
}

// non-optional because the implementation makes it a bit hard to check if the file exists
// (vs. Docker run error), and our current callsite needs non-optional anyway
func loadNonOptionalBaseImageConf(builder bobfile.BuilderSpec) (*BaseImageConfig, error) {
	dockerImage, err := func() (string, error) {
		kind, data, err := parseBuilderUsesType(builder.Uses)
		if err != nil {
			return "", err
		}

		//nolint:exhaustive
		switch kind {
		case builderUsesTypeImage:
			return data, nil
		default:
			return "", errors.New("cannot load base image config from non-Docker-image builder")
		}
	}()
	if err != nil {
		return nil, err
	}

	// unfortunately there isn't a good high-level way to grab a file from a Docker image, so that's
	// why we have to create a container to get it
	content, err := exec.Command("docker", "run", "--rm", dockerImage, "cat", baseImageJsonLocation).Output()
	if err != nil {
		return nil, err
	}

	conf := &BaseImageConfig{}
	if err := jsonfile.UnmarshalDisallowUnknownFields(bytes.NewReader(content), conf); err != nil {
		return nil, err
	}

	baseImageConfMigrate(conf) // mutates

	return conf, nil
}

func baseImageConfMigrate(conf *BaseImageConfig) {
	// upgrade Deprecated1 to LangserverSpec
	if len(conf.Deprecated1) > 0 {
		conf.Langserver = &LangserverSpec{
			Command:   conf.Deprecated1,
			Languages: []string{"go"}, // AFAIK the only user was buildkit-golang
		}

		conf.Deprecated1 = nil
	}
}
