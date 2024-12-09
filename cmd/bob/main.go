package main

import (
	"fmt"
	"os"

	"github.com/function61/gokit/app/dynversion"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/powerline"
	"github.com/spf13/cobra"
)

/*	Process

	- Check out correct revision from version control
	- Build all build containers, without context
	- Run build containers, in correct order, to produce artefacts
	- Run "after build" steps

	Environment

	- All build containers share /build, /tmp

	Build results:

	- (HOST) Push built Docker container
	- (CONTAINER) Custom actions done inside the build container (publish compiled binaries to S3 etc.)
	- Any combination of those
*/

func printHeading(content string) {
	fmt.Printf("\n====== %s\n", content)
}

func main() {
	// we've to init root Cobra command here (and not in global scope as examples would suggest),
	// not only because this is cleaner but also because reference to global "version" var would
	// refer to the "dev" value even when overriding it from build command. I don't know why it works like that.
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Turbo Bob (the builder) helps you build and develop your projects.",
		Version: dynversion.Version,
	}

	inside, err := insideDevContainer()
	osutil.ExitIfError(err)

	if !inside {
		app.AddCommand(buildEntry())
		app.AddCommand(devEntry())
		app.AddCommand(infoEntry())
		app.AddCommand(workspaceEntry())

		app.AddCommand(openProjectHomepageEntrypoint())

		// hidden "$ bob init" for convenience (it exists as non-hidden under tools)
		app.AddCommand(initEntryWithHidden(true))

		app.AddCommand(toolsEntry()) // namespace for less often needed tools
	} else {
		app.AddCommand(buildInsideEntry())
		app.AddCommand(tipsEntry())

		// below are never visible, internal-use only commands
		app.AddCommand(powerline.Entrypoint())
		app.AddCommand(devShimEntry())
	}

	// these commands are visible from both inside and outside
	app.AddCommand(triggerEntry())

	osutil.ExitIfError(app.Execute())
}

// tools namespace because we don't want to pollute the root namespace (meant for fast discovery of
// everyday commands) with tens of less often used commands.
func toolsEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Less often needed tools",
	}

	cmd.AddCommand(smokeTestEntrypoint())

	// TODO: move powerline here?
	cmd.AddCommand(initEntry())
	cmd.AddCommand(langserverEntry())

	return cmd
}

func insideDevContainer() (bool, error) {
	return osutil.Exists(shimDataDirContainer)
}
