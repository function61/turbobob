package main

// Support for making langserver (https://langserver.org/) integration easier and better with containers.

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/spf13/cobra"
)

func langserverEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "langserver",
		Short: "Launch a langserver process. Must be run in project's root dir. Intended to be run by your editor.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(langserverRunShim(
				osutil.CancelOnInterruptOrTerminate(nil)))
		},
	}

	return cmd
}

// automatically detects which project this invocation concerns
func langserverRunShim(ctx context.Context) error {
	// access chosen project's details (so we know which langserver to start)
	projectFile, err := bobfile.Read()
	if err != nil {
		return err
	}

	langserverCmd, builder, err := func() ([]string, *bobfile.BuilderSpec, error) {
		for _, builder := range projectFile.Builders {
			// FIXME: this assumes all builders have a config file defined
			baseImageConf, err := loadNonOptionalBaseImageConf(builder)
			if err != nil {
				return nil, nil, fmt.Errorf("loadNonOptionalBaseImageConf: %w", err)
			}

			if baseImageConf.Langserver == nil {
				continue
			}

			return baseImageConf.Langserver.Command, &builder, nil
		}

		return nil, nil, fmt.Errorf("%s doesn't define a compatible language server", projectFile.ProjectName)
	}()
	if err != nil {
		return err
	}

	containerName := devContainerName(projectFile, *builder)

	if !isDevContainerRunning(containerName) {
		return fmt.Errorf("container '%s' is not running. did you forget to run `$ bob dev` first?", containerName)
	}

	// not using "--tty" because with it we got "gopls: the input device is not a TTY"
	dockerized := append([]string{"docker", "exec", "--interactive", containerName}, langserverCmd...)

	//nolint:gosec // ok
	langserver := exec.CommandContext(ctx, dockerized[0], dockerized[1:]...)
	langserver.Stdin = os.Stdin
	langserver.Stdout = os.Stdout
	langserver.Stderr = os.Stderr

	return langserver.Run()
}
