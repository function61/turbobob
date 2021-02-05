package main

import (
	"fmt"
	"os"

	"github.com/function61/gokit/dynversion"
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
		app.AddCommand(initEntry())
		app.AddCommand(buildEntry())
		app.AddCommand(devEntry())
		app.AddCommand(infoEntry())

		app.AddCommand(workspaceEntry())
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

func insideDevContainer() (bool, error) {
	return osutil.Exists(shimDataDirContainer)
}
