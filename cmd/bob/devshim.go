package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

const (
	shimDataDirContainer = "/tmp/bob-shim"
	shimConfigFile       = "shim_config.json"
)

type shimConfig struct {
	BuilderName               string   `json:"builder_name"` // which builder we're inside in. required to resolve which command "bob build" should run
	DynamicProTipsFromHost    []string `json:"dynamic_pro_tips_from_host"`
	EnablePromptCustomization bool     `json:"enable_prompt_customization"`
}

// dev shim is used when as entry point for "$ bob dev" in container's side to set up some
// usability tricks (like shell history injection) before starting the shell

func devShimEntry() *cobra.Command {
	return &cobra.Command{
		Use:    "dev-shim",
		Short:  "Shim on the container side",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1), // at least the process to start
		Run: func(cmd *cobra.Command, args []string) {
			shimExitIfErr("shimSetup", shimSetup())

			// we came here from "$ docker run <image> ..."
			// you can do "$ docker run <image> sh" and it works, but we can't syscall.Exec("sh")
			// but we have to give sh's full path instead. so we'll try to resolve it here.
			fullPath, err := exec.LookPath(args[0])
			shimExitIfErr("LookPath", err)

			envs := os.Environ()
			if filepath.Base(args[0]) == "sh" { // accept both "/bin/sh", "sh"
				// qualified path to .bashrc because "sh" on Alpine doesn't accept ~/.bashrc
				// from us, but does accept interactively
				envs = append(envs, "ENV=/root/.bashrc")
			}

			// stdout seems to buffer in such a way that when we Exec(), the last line gets
			// lost. this is to mitigate it so that if a line is ignored, it'll be this one.
			// https://eklitzke.org/stdout-buffering
			// apparently, Exec() is even discouraged: https://go-review.googlesource.com/c/go/+/72550/
			fmt.Println("")

			// on successful start, this never returns.
			// can take either absolute or relative (exec.LookPath() could give relative) path:
			//   https://stackoverflow.com/q/33852690/2176740
			shimExitIfErr("exec", syscall.Exec(fullPath, args, envs))
		},
	}
}

func shimSetup() error {
	// some things must not be done again
	alreadyDone, err := osutil.Exists(shimSetupDoneFlagPath())
	if err != nil {
		return err
	}

	bobfile, err := readBobfile()
	if err != nil {
		return fmt.Errorf("readBobfile: %w", err)
	}

	shimConf, err := readShimConfig()
	if err != nil {
		return fmt.Errorf("readShimConfig: %w", err)
	}

	builder, err := findBuilder(bobfile, shimConf.BuilderName)
	if err != nil {
		return fmt.Errorf("findBuilder: %w", err)
	}

	baseImgConf, err := loadBaseImageConfWhenInsideContainer()
	if err != nil {
		return err
	}

	for _, pathToCache := range baseImgConf.PathsToCache {
		if err := makeCacheDir(pathToCache); err != nil {
			return fmt.Errorf("makeCacheDir: %w", err)
		}
	}

	if err := injectHistory(*builder, alreadyDone, *baseImgConf); err != nil {
		return fmt.Errorf("injectHistory: %w", err)
	}

	if shimConf.EnablePromptCustomization && !alreadyDone {
		if err := customizePrompt(); err != nil {
			return fmt.Errorf("customizePrompt: %w", err)
		}
	}

	if !alreadyDone {
		// set up flag, so the next session logging in to this container will not try to
		// do setup steps that must be done only once
		if err := makeEmptyFile(shimSetupDoneFlagPath()); err != nil {
			return err
		}
	}

	fmt.Println("Pro tips: (see '$ bob tips' for all)")

	printProTip := func(proTip string) {
		fmt.Println("   " + proTip)
	}

	// important tips are to be shown automatically on container entering
	for _, command := range baseImgConf.DevShellCommands {
		if command.Important {
			printProTip("$ " + command.Command)
		}
	}

	for _, command := range builder.DevShellCommands {
		if command.Important {
			printProTip("$ " + command.Command)
		}
	}

	for _, proTip := range builder.DevProTips {
		printProTip(proTip)
	}

	for _, proTip := range shimConf.DynamicProTipsFromHost {
		printProTip(proTip)
	}

	return nil
}

// so we can recall frequently needed commands fast + sets up HISTFILE ENV var
func injectHistory(builder BuilderSpec, alreadyDone bool, baseImgConf BaseImageConfig) error {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	histfilePath := filepath.Join(userHomeDir, ".bash_history")

	// at least for Alpine Linux's /bin/sh this is not set and used without setting this.
	// fix this before maybe-return-on-optional-file-not-existing
	if err := os.Setenv("HISTFILE", histfilePath); err != nil {
		return err
	}

	if alreadyDone { // we've seen this before. don't append dev shell history
		return nil
	}

	histfile, err := os.OpenFile(
		histfilePath,
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0600)
	if err != nil {
		return err
	}
	defer histfile.Close()

	historyToAdd := append(
		append(allDevShellCommands(builder.DevShellCommands), allDevShellCommands(baseImgConf.DevShellCommands)...),
		"bob tips", // add a few builtin commands which all builders share
		"bob build --fast",
		"bob build")

	_, err = io.Copy(histfile, strings.NewReader(strings.Join(historyToAdd, "\n")+"\n"))
	return err
}

// makes prettier prompt
func customizePrompt() error {
	// instructs Bash to ask bob to generate the prompt string on each time the prompt is needed.
	// Bob makes a pretty Powerline-inspired prompt (https://github.com/powerline/powerline)
	return ioutil.WriteFile("/root/.bashrc", []byte(`
# customization written by Turbo Bob
export PS1="\$(bob powerline \$?)"
`), 0755)
}

func shimExitIfErr(prefix string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "(shim) %s: %v\n", prefix, err)
		os.Exit(1)
	}
}

// make various dirs symlinks to a bind mount from host, so cache can be shared between
// multiple container instances
func makeCacheDir(dir string) error {
	exists, err := osutil.Exists(dir)
	if err != nil {
		return err
	}

	if exists { // only symlink if path does not exist
		return nil // not an error per se
	}

	// before symlink succeeds, parent must exist (this is no-op if already exists)
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return err
	}

	// "/go/pkg" => "/tmp/build/go/pkg"
	cacheCounterpart := filepath.Join("/tmp/build/", dir)

	if err := os.MkdirAll(cacheCounterpart, 0755); err != nil {
		return err
	}

	return os.Symlink(cacheCounterpart, dir)
}

func readShimConfig() (*shimConfig, error) {
	shimConf := &shimConfig{}
	return shimConf, jsonfile.ReadDisallowUnknownFields(filepath.Join(shimDataDirContainer, shimConfigFile), shimConf)
}

func shimSetupDoneFlagPath() string {
	return filepath.Join(os.TempDir(), "bob-shim-setup-done.flag")
}

// "$ touch"
func makeEmptyFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return f.Close()
}
