package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var version = "dev"

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

var rootCmd = &cobra.Command{
	Use:     os.Args[0],
	Short:   "Turbo Bob (the builder) helps you build and develop your projects.",
	Version: version,
}

func printHeading(content string) {
	fmt.Printf("\n====== %s\n", content)
}

func main() {
	rootCmd.AddCommand(initEntry())
	rootCmd.AddCommand(buildEntry())
	rootCmd.AddCommand(devEntry())
	rootCmd.AddCommand(infoEntry())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func reactToError(err error) {
	if err != nil {
		panic(err)
	}
}
