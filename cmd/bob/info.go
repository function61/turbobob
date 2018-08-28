package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func info() error {
	bobfile, errBobfile := readBobfile()
	if errBobfile != nil {
		return errBobfile
	}

	metadata, errMetadata := resolveMetadataFromVersionControl()
	if errMetadata != nil {
		return errMetadata
	}

	fmt.Printf("Project name: %s\n", bobfile.ProjectName)
	fmt.Printf("VcKind: %s\n", metadata.VcKind)
	fmt.Printf("Revision ID (full): %s (%s)\n", metadata.RevisionIdShort, metadata.RevisionId)
	fmt.Printf("Friendly revision: %s\n", metadata.FriendlyRevisionId)

	for _, builder := range bobfile.Builders {
		fmt.Printf("\n----\nBuilder: %s\n", builder.Name)
		fmt.Printf("Mount dir: %s\n", builder.MountDestination)
		for _, envKey := range builder.PassEnvs {
			setOrNot := "NOT SET"
			if os.Getenv(envKey) != "" {
				setOrNot = "SET"
			}

			fmt.Printf("ENV: %s (%s)\n", envKey, setOrNot)
		}
	}

	return nil
}

func infoEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Displays info about the project",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(info())
		},
	}
}
