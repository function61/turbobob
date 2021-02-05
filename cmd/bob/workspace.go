package main

// Workspace management commands, with assumption that you use one window manager workspace
// ("virtual desktop") per project. With these commands you can "$ cd" to your project, say
// "$ bob ws edit --rename" to launch code editor for your project and rename your window manager's
// workspace according to the project.
//
// Currently assumes you're using i3

import (
	"os"

	"github.com/function61/gokit/os/osutil"
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
	workdir, err := os.Getwd()
	if err != nil {
		return err
	}

	userConfig, err := loadUserconfigFile()
	if err != nil {
		return err
	}

	codeEditorCmd, err := userConfig.CodeEditorCmd(workdir)
	if err != nil {
		return err
	}

	return resolveWindowManager().LaunchDisownedProgram(codeEditorCmd...)
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
	bobfile, err := readBobfile()
	if err != nil {
		return err
	}

	return resolveWindowManager().RenameCurrentWorkspace(bobfile.ProjectName)
}
