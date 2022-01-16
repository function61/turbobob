package main

// Support for making langserver (https://langserver.org/) integration easier and better with containers.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

func langserverEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "langserver",
		Short: "Launch a langserver process. Must be run in project's root dir. Intended to be run by your editor.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(langserverRunShim(
				osutil.CancelOnInterruptOrTerminate(nil)))
		},
	}
}

// automatically detects which project this invocation concerns
func langserverRunShim(ctx context.Context) error {
	workdir, err := resolveWorkdirFromLSInvocation()
	if err != nil {
		return err
	}

	// workdir is not always the same as process's initial workdir.
	// after this, we can resolve the chosen project's details.
	if err := os.Chdir(workdir); err != nil {
		return err
	}

	// in Bob dev containers we might have /workspace mount (i.e. different mount point than source
	// path in host), but editors send file references to LS's with the paths they're seeing, so we
	// must use the same path in containers (unless we want to do tricks with symlinks etc.)
	mountDir := workdir

	// access chosen project's details (so we know which programming language's langserver to start)
	bobfile, err := readBobfile()
	if err != nil {
		return err
	}

	if len(bobfile.Builders) < 1 {
		return errors.New("need at least one builder")
	}

	// FIXME: this is not correct. could use builder.MountSource to resolve specific one
	builder := bobfile.Builders[0]

	langserverCmd, err := func() ([]string, error) {
		baseImageConf, err := loadNonOptionalBaseImageConf(*bobfile, builder)
		if err != nil {
			return nil, err
		}

		if len(baseImageConf.LangserverCmd) == 0 {
			return nil, fmt.Errorf("%s doesn't define a language server", builder.Uses)
		}

		return baseImageConf.LangserverCmd, nil
	}()
	if err != nil {
		return err
	}

	kind, dockerImage, err := parseBuilderUsesType(builder.Uses)
	if err != nil || kind != builderUsesTypeImage {
		return fmt.Errorf("not Docker image or failure parsing uses: %w", err)
	}

	// not using "--tty" because with it we got "gopls: the input device is not a TTY"
	dockerized := append([]string{"docker", "run",
		"--rm",          // so resources get released. (this process is ephemeral in nature)
		"--interactive", // use stdin (it is the transport for one direction in LSP)
		"--name=" + langServerContainerName(bobfile, builder),
		"--volume", fmt.Sprintf("%s:%s", workdir, mountDir),
		dockerImage,
	}, langserverCmd...)

	//nolint:gosec // ok
	langserver := exec.CommandContext(ctx, dockerized[0], dockerized[1:]...)
	langserver.Stdin = os.Stdin
	langserver.Stdout = os.Stdout
	langserver.Stderr = os.Stderr

	return langserver.Run()
}

// we can't use os.Getwd() directly, because at least Sublime Text resolves symlink-based locations
// to their actual locations, and if we were to map that location inside LS container, the editor
// would communicate different (not visible) file paths to the container
func resolveWorkdirFromLSInvocation() (string, error) {
	// our default workdir is the process' one
	workdirDefault, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// TODO: read this from Bob's userconfig
	wrongAndCorrectWds := map[string]string{
		"/persist/work": "/home/joonas/work",
	}

	for wdWrong, wdCorrect := range wrongAndCorrectWds {
		if strings.HasPrefix(workdirDefault, wdWrong) {
			/*	given:
				wdWrong=/wrong/work
				wdCorrect=/correct/work

				translates /wrong/work/projectx/file_y.go -> /correct/work/projectx/file_y.go
			*/
			translated := wdCorrect + strings.TrimPrefix(workdirDefault, wdWrong)

			log.Printf("translated incorrect prefix %s to %s", wdWrong, translated)

			return translated, nil
		}
	}

	// no correction had to be made, so workdir was already correct
	return workdirDefault, nil
}
