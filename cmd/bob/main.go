package main

import (
	"fmt"
	"os"
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

func usage(selfName string) error {
	fmt.Printf("Usage: %s\n", os.Args[0])
	fmt.Printf("\tinfo\n")
	fmt.Printf("\tbuild <revision>\n")
	fmt.Printf("\tbuild-in-ci\n")
	fmt.Printf("\tdev <builder>\n")

	return nil
}

func printHeading(content string) {
	fmt.Printf("# %s\n", content)
}

func main() {
	mainInternal := func() error {
		if len(os.Args) < 2 {
			return usage(os.Args[0])
		}

		bobfile, errBobfile := readBobfile()
		if errBobfile != nil {
			if os.IsNotExist(errBobfile) {
				return ErrBobfileNotFound
			}

			return errBobfile
		}

		switch os.Args[1] {
		case "build-in-ci":
			return buildInCi(bobfile)
		case "build":
			return build(bobfile)
		case "dev":
			return dev(bobfile, os.Args[2:])
		case "info":
			return info(bobfile)
		/* case "init":
		return init() */
		default:
			return unknownCommand(os.Args[1])
		}
	}

	if err := mainInternal(); err != nil {
		panic(err)
	}
}
