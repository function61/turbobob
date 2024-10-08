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
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/spf13/cobra"
)

func langserverEntry() *cobra.Command {
	lang := ""

	cmd := &cobra.Command{
		Use:   "langserver",
		Short: "Launch a langserver process. Must be run in project's root dir. Intended to be run by your editor.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(langserverRunShim(
				osutil.CancelOnInterruptOrTerminate(nil),
				strings.Split(lang, ",")))
		},
	}

	cmd.Flags().StringVarP(&lang, "lang", "", lang, "Language")

	return cmd
}

// automatically detects which project this invocation concerns
func langserverRunShim(ctx context.Context, langs []string) error {
	if len(langs) == 0 || langs[0] == "" {
		return errors.New("--lang not specified")
	}

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
	projectFile, err := bobfile.Read()
	if err != nil {
		return err
	}

	langserverCmd, builder, err := func() ([]string, *bobfile.BuilderSpec, error) {
		for _, builder := range projectFile.Builders {
			// FIXME: this assumes
			baseImageConf, err := loadNonOptionalBaseImageConf(builder)
			if err != nil {
				return nil, nil, fmt.Errorf("loadNonOptionalBaseImageConf: %w", err)
			}

			if baseImageConf.Langserver == nil {
				continue
			}

			if anyOfLanguagesMatch(langs, baseImageConf.Langserver.Languages) {
				return baseImageConf.Langserver.Command, &builder, nil
			}
		}

		return nil, nil, fmt.Errorf(
			"%s doesn't define a compatible language server for %v",
			projectFile.ProjectName,
			langs)
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
		"--rm", // so resources get released. (this process is ephemeral in nature)
		"--shm-size=512M",
		"--interactive", // use stdin (it is the transport for one direction in LSP)
		"--name=" + langServerContainerName(projectFile, *builder),
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

func anyOfLanguagesMatch(want []string, got []string) bool {
	for _, wantItem := range want {
		if sliceutil.ContainsString(got, wantItem) {
			return true
		}
	}

	return false
}
