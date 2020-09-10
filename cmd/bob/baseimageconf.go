package main

import (
	"fmt"

	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/jsonfile"
)

const (
	baseImageJsonLocation = "/turbobob-baseimage.json"
)

// base image conf JSON - able to provide hints for useful commands, setting up cache paths etc.
type BaseImageConfig struct {
	DevShellCommands []DevShellCommand `json:"dev_shell_commands"`
	PathsToCache     []string          `json:"paths_to_cache"` // will be set up as symlinks to a persistent mountpoint, so that subsequent containers benefit from cache

	FileDescriptionBoilerplate string `json:"for_description_of_this_file_see"` // URL to Bob homepage
}

// base image conf is optional. if it doesn't exist, an empty (but valid) conf will be
// returned without error
func loadBaseImageConf() (*BaseImageConfig, error) {
	exists, err := fileexists.Exists(baseImageJsonLocation)
	if err != nil {
		return nil, fmt.Errorf("loadBaseImageConf: %w", err)
	}

	if !exists {
		return &BaseImageConfig{}, nil
	}

	conf := &BaseImageConfig{}
	if err := jsonfile.Read(baseImageJsonLocation, conf, true); err != nil {
		return nil, fmt.Errorf("loadBaseImageConf: %w", err)
	}

	return conf, nil
}
