package main

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/jsonfile"
	"os"
	"path/filepath"
)

type UserconfigFile struct {
	DevIngressSettings devIngressSettings `json:"dev_ingress_settings"`
}

type devIngressSettings struct {
	Domain        string `json:"domain"`  // app ID "foo" with domain "example.com" will be exposed at foo.example.com
	DockerNetwork string `json:"network"` // optional - if you want to place dev containers on a specific Docker network
}

func (d devIngressSettings) Validate() error {
	if d.Domain == "" {
		return errors.New("Domain must not be empty")
	}

	return nil
}

// if not found, returns default
func loadUserconfigFile() (*UserconfigFile, error) {
	userHomedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("loadUserconfigFile: %w", err)
	}

	confFilePath := filepath.Join(userHomedir, "turbobob-userconfig.json")

	exists, err := fileexists.Exists(confFilePath)
	if err != nil {
		return nil, fmt.Errorf("loadUserconfigFile: %w", err)
	}

	conf := &UserconfigFile{}

	if exists {
		return conf, maybeWrapErr("loadUserconfigFile: %w", jsonfile.Read(confFilePath, conf, true))
	} else {
		// return empty struct, because it's fully valid to use Turbo Bob without
		// user-specific config
		return conf, nil
	}
}

func maybeWrapErr(formatString string, err error) error {
	if err != nil {
		return fmt.Errorf(formatString, err)
	}

	return nil
}
