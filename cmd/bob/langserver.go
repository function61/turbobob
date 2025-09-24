package main

// Support for making langserver (https://langserver.org/) integration easier and better with containers.

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func langserverEntry() *cobra.Command {
	lang := ""
	langGeneric := ""

	cmd := &cobra.Command{
		Use:   "langserver",
		Short: "Launch a langserver process. Must be run in project's root dir. Intended to be run by your editor.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			if langGeneric != "" { // explicit requested language (not defined by project)
				osutil.ExitIfError(langserverRunGeneric(osutil.CancelOnInterruptOrTerminate(nil), langGeneric))
				return
			} else {
				// project's defined langserver
				osutil.ExitIfError(langserverRunShim(
					osutil.CancelOnInterruptOrTerminate(nil),
					lang))

			}
		},
	}

	cmd.Flags().StringVarP(&lang, "lang-from-project", "", lang, "Language (go / ts / ...) whose language server to start")
	cmd.Flags().StringVarP(&langGeneric, "lang-generic", "", langGeneric, "Generic (= defined outside of project) language server to start")

	return cmd
}

// automatically detects which project this invocation concerns.
// if `langCodeRequested` empty, accepts first language server that is found from builders.
func langserverRunShim(ctx context.Context, langCodeRequested string) error {
	// access chosen project's details (so we know which langserver to start)
	projectFile, err := bobfile.Read()
	if err != nil {
		return err
	}

	langserverCmd, builder, err := func() ([]string, *bobfile.BuilderSpec, error) {
		for _, builder := range projectFile.Builders {
			// FIXME: this assumes all builders have a config file defined
			baseImageConf, err := loadNonOptionalBaseImageConf(projectFile.ProjectName, builder)
			if err != nil {
				return nil, nil, fmt.Errorf("loadNonOptionalBaseImageConf: %w", err)
			}

			lsDetails := baseImageConf.Langserver
			languageMatches := langCodeRequested == "" || lo.Contains(lsDetails.Languages, langCodeRequested)
			if lsDetails == nil || !languageMatches {
				continue
			}

			return lsDetails.Command, &builder, nil
		}

		return nil, nil, fmt.Errorf("%s doesn't define a compatible language server", projectFile.ProjectName)
	}()
	if err != nil {
		return err
	}

	// LSP process needs to run in the dev container (not a separate langserver container) because it might need access
	// to compiler cache, built object files etc.
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

func langserverRunGeneric(ctx context.Context, language string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	type langServerImage struct {
		ref        string
		args       []string
		dockerOpts []string
	}

	// NOTE: need for `--clientProcessId=1` see https://github.com/denoland/deno/issues/22012
	//
	// more idiocy: https://github.com/microsoft/vscode-languageserver-node/blob/d58c00bbf8837b9fd0144924db5e7b1c543d839e/server/src/node/main.ts#L78-L104
	knownGenericServers := map[string]langServerImage{
		"bash": {ref: "ghcr.io/r-xs-fi/bash-language-server", args: []string{"start", "--clientProcessId=1"}},
		"json": {ref: "ghcr.io/r-xs-fi/vscode-langservers-extracted", args: []string{"vscode-json-language-server", "--stdio", "--clientProcessId=1"}},
		"css":  {ref: "ghcr.io/r-xs-fi/vscode-langservers-extracted", args: []string{"vscode-css-language-server", "--stdio", "--clientProcessId=1"}},
		"html": {ref: "ghcr.io/r-xs-fi/vscode-langservers-extracted", args: []string{"vscode-html-language-server", "--stdio", "--clientProcessId=1"}},
		// TODO: add markdown server when upstream project supports it
	}

	server, found := knownGenericServers[language]
	if !found {
		return fmt.Errorf("generic language server not found for: %s", language)
	}

	args := []string{"docker", "run", "--rm", "--interactive", "--workdir=" + wd, "--volume=" + wd + ":" + wd}
	args = append(args, server.dockerOpts...)
	args = append(args, server.ref)
	args = append(args, server.args...)

	//nolint:gosec // ok
	lspServer := exec.CommandContext(ctx, args[0], args[1:]...)
	/*
		if strace := false; strace {
			// TODO: don't limit string lengths
			args = append(args, "strace")

			if followForks := false; followForks {
				args = append(args, "--follow-forks")
			}
		}
		args = append(args, extraArgs...)

		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdin = io.TeeReader(os.Stdin, lspstdin)
		cmd.Stdout = io.MultiWriter(os.Stdout, lsplog)
		// cmd.Stderr = os.Stderr
		cmd.Stderr = lsplogStderr
	*/
	lspServer.Stdin = os.Stdin
	lspServer.Stdout = os.Stdout
	lspServer.Stderr = os.Stderr // not used officially by LSP, but good to have errors from Docker etc. that could be shown in error logs of LSP client

	return lspServer.Run()
}
