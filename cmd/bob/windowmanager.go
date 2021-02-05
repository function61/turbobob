package main

// Bob can use some optional help from window managers like rename current workspace according to our project.

import (
	"errors"
	"fmt"
	"strings"

	"go.i3wm.org/i3/v4"
)

// stuff window managers can do
type WindowManager interface {
	RenameCurrentWorkspace(name string) error
	// disowned = our process is not the parent, so the process's lifetime is not tied to our process' lifetime.
	// TODO: there might be a generic way to disown without a dependency to chosen WM
	LaunchDisownedProgram(cmd ...string) error
}

func resolveWindowManager() WindowManager {
	// TODO: don't assume - it'd be cleaner to get this from user's config file
	return &i3wm{}
}

type i3wm struct{}

var _ WindowManager = (*i3wm)(nil)

func (i *i3wm) RenameCurrentWorkspace(name string) error {
	focusedWorkspace, err := findi3FocusedWorkspace()
	if err != nil {
		return err
	}

	// i3's workspace names contain their order number as well
	newWorkspaceName := func() string {
		if focusedWorkspace.Num != -1 { // is possible
			return fmt.Sprintf("%d %s", focusedWorkspace.Num, name)
		} else {
			return name
		}
	}()

	_, err = i3.RunCommand(fmt.Sprintf(
		`rename workspace "%s" to "%s"`,
		i3EscapeQuotes(focusedWorkspace.Name),
		i3EscapeQuotes(newWorkspaceName)))
	return err
}

func (i *i3wm) LaunchDisownedProgram(cmd ...string) error {
	_, err := i3.RunCommand(fmt.Sprintf(`exec "%s"`, i3EscapeQuotes(strings.Join(cmd, " "))))
	return err
}

func findi3FocusedWorkspace() (*i3.Workspace, error) {
	workspaces, err := i3.GetWorkspaces()
	if err != nil {
		return nil, err
	}

	for _, ws := range workspaces {
		if ws.Focused {
			return &ws, nil
		}
	}

	return nil, errors.New("unable to find focused workspace") // should not happen
}

func i3EscapeQuotes(input string) string {
	return strings.ReplaceAll(input, `"`, `\"`)
}
