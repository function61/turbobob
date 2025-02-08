package main

// Workspace management commands, with assumption that you use one window manager workspace
// ("virtual desktop") per project. With these commands you can "$ cd" to your project, say
// "$ bob ws edit --rename" to launch code editor for your project and rename your window manager's
// workspace according to the project.
//
// Currently assumes you're using i3

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"al.essio.dev/pkg/shellescape"
	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/spf13/cobra"
)

func workspaceEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ws",
		Short: "Workspace related commands",
	}

	renameWorkspace := false
	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Launch code editor",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(workspaceLaunchEditor())

			if renameWorkspace {
				osutil.ExitIfError(workspaceRenameToSelectedProject())
			}
		},
	}
	editCmd.Flags().BoolVarP(&renameWorkspace, "rename", "r", renameWorkspace, "Rename window manager workspace to project")
	cmd.AddCommand(editCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "browse",
		Short: "Open file browser",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(workspaceLaunchFileBrowser())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "terminal [terminal]",
		Short: "Wrapper for launching a terminal in the currently focused workspace's project directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(workspaceLaunchTerminal(args[0]))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "rename",
		Short: "Rename window manager workspace to project",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(workspaceRenameToSelectedProject())
		},
	})

	return cmd
}

func workspaceLaunchEditor() error {
	withErr := func(err error) error { return fmt.Errorf("workspaceLaunchEditor: %w", err) }

	workdir, err := os.Getwd()
	if err != nil {
		return withErr(err)
	}

	userConfig, err := loadUserconfigFile()
	if err != nil {
		return withErr(err)
	}

	codeEditorCmd, err := userConfig.CodeEditorCmd(workdir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) { // none specified
			editor := discoverSensibleEditor()

			codeEditorCmd = []string{editor, workdir}
		} else {
			return withErr(err)
		}
	}

	if err := resolveWindowManager().LaunchDisownedProgram("i3-sensible-terminal", "-e", shellescape.QuoteCommand(codeEditorCmd)); err != nil {
		return withErr(err)
	}

	return nil
}

// try to use some (standards-compliant) method to guess user's preferred editor.
// in the spirit of https://github.com/i3/i3/blob/next/i3-sensible-editor
func discoverSensibleEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// /usr/bin/editor from Debian-alternatives (i.e. user's chosen editor)
	return "editor"
}

func workspaceLaunchFileBrowser() error {
	workdir, err := os.Getwd()
	if err != nil {
		return err
	}

	userConfig, err := loadUserconfigFile()
	if err != nil {
		return err
	}

	cmd, err := userConfig.FileBrowserCmd(workdir)
	if err != nil {
		return err
	}

	return resolveWindowManager().LaunchDisownedProgram(cmd...)
}

func workspaceRenameToSelectedProject() error {
	withErr := func(err error) error { return fmt.Errorf("workspaceRenameToSelectedProject: %w", err) }

	bobfile, err := bobfile.Read()
	if err != nil {
		return withErr(err)
	}

	userConfig, err := loadUserconfigFile()
	if err != nil {
		return withErr(err)
	}

	projectNameWithMaybeIcon := func() string {
		if projectEmojiIcon := bobfile.ProjectEmojiIcon(); projectEmojiIcon != "" && userConfig.WindowManagerShowProjectEmojiIcons {
			return fmt.Sprintf("%s %s", projectEmojiIcon, bobfile.ProjectName)
		} else {
			return bobfile.ProjectName
		}
	}()

	workdir, err := os.Getwd()
	if err != nil {
		return withErr(err)
	}

	reg, err := readWorkspaceNameRegistry()
	if err != nil {
		return withErr(err)
	}
	reg[projectNameWithMaybeIcon] = workdir // record to registry

	if err := jsonfile.Write(workspaceNameRegistryPath, reg); err != nil {
		return withErr(err)
	}
	if err := resolveWindowManager().RenameCurrentWorkspace(projectNameWithMaybeIcon); err != nil {
		return withErr(err)
	}

	return nil
}

func workspaceLaunchTerminal(terminal string) error {
	focusedWorkspace, err := findi3FocusedWorkspace()
	if err != nil {
		return err
	}

	// our window manager interfacing code added workspace number to the workspace name so to match on the name
	// we need to remove the number first. this is a bit hacky.
	//
	prefixToTrim := fmt.Sprintf("%d ", focusedWorkspace.Num)
	focusedWorkspaceNameWithoutNumber := strings.TrimPrefix(focusedWorkspace.Name, prefixToTrim)

	workspaceNameToProjectRoot, err := readWorkspaceNameRegistry()
	if err != nil {
		return err
	}

	terminalEnvironmentToUse := func() []string {
		projectRoot, found := workspaceNameToProjectRoot[focusedWorkspaceNameWithoutNumber]
		// projectRoot, found := "/home/joonas/work/turbobob", true
		if found { // not found case is ok as well, we simply don't do pwd manipulation
			if err := os.Chdir(projectRoot); err != nil {
				panic(err)
			}
			// can't do only os.Chdir() here as it manipulates the `PWD` env variable I guess, but `os.Environ()`
			// is cached and therefore it would not get reflected to `syscall.Exec()`
			//
			//TODO: find out why both os.Chdir() and ENV tuning needed!!!
			return append(os.Environ(), "PWD="+projectRoot)
		} else {
			return os.Environ()
		}
	}()

	terminalAbsolutePath, err := exec.LookPath(terminal) // exec needs absolute path
	if err != nil {
		return err
	}
	// using exec to spare one unnecessary hangaround process.
	// NOTE: this is not expected to return in the happy path.
	if err := syscall.Exec(terminalAbsolutePath, nil, terminalEnvironmentToUse); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

const (
	workspaceNameRegistryPath = "/tmp/bob-workspaces.json"
)

// maps workspace name => workspace project root
type workspaceNameRegistry map[string]string

func readWorkspaceNameRegistry() (workspaceNameRegistry, error) {
	reg := workspaceNameRegistry{}
	if err := jsonfile.ReadDisallowUnknownFields(workspaceNameRegistryPath, &reg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return reg, err
	}
	return reg, nil
}
