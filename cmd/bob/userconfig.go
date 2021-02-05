package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
)

type programConfig struct {
	Cmd []string `json:"cmd"`
}

type UserconfigFile struct {
	DevIngressSettings        devIngressSettings `json:"dev_ingress_settings"`
	EnablePromptCustomization *bool              `json:"enable_prompt_customization"`
	CodeEditor                *programConfig     `json:"code_editor"`  // .cmd can contain "$PROJECT_ROOT" if you need to path to project as arg
	FileBrowser               *programConfig     `json:"file_browser"` // .cmd can contain "$DIRECTORY" if your file browser doesn't use its workdir
}

func (u *UserconfigFile) CodeEditorCmd(projectRoot string) ([]string, error) {
	if u.CodeEditor == nil {
		return nil, errors.New("code editor not specified in user config file")
	}

	cmd := []string{}
	for _, cmdPart := range u.CodeEditor.Cmd {
		cmd = append(cmd, strings.ReplaceAll(cmdPart, "$PROJECT_ROOT", projectRoot))
	}

	return cmd, nil
}

func (u *UserconfigFile) FileBrowserCmd(directory string) ([]string, error) {
	if u.FileBrowser == nil {
		return nil, errors.New("file browser not specified in user config file")
	}

	cmd := []string{}
	for _, cmdPart := range u.FileBrowser.Cmd {
		cmd = append(cmd, strings.ReplaceAll(cmdPart, "$DIRECTORY", directory))
	}

	return cmd, nil
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

// if not found, returns default values
func loadUserconfigFile() (*UserconfigFile, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("loadUserconfigFile: %w", err)
	}

	confFilePath := filepath.Join(userConfigDir, "turbobob", "config.json")

	exists, err := osutil.Exists(confFilePath)
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
