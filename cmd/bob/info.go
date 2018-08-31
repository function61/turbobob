package main

import (
	"fmt"
	"github.com/apcera/termtables"
	"github.com/spf13/cobra"
	"os"
	"strings"
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

	basicDetails := termtables.CreateTable()
	basicDetails.AddRow("Project name", bobfile.ProjectName)
	basicDetails.AddRow("VcKind", metadata.VcKind)
	basicDetails.AddRow("Revision ID (full)", fmt.Sprintf("%s (%s)", metadata.RevisionIdShort, metadata.RevisionId))
	basicDetails.AddRow("Friendly revision", metadata.FriendlyRevisionId)

	fmt.Printf("BASIC DETAILS\n%s\n", basicDetails.Render())

	for _, builder := range bobfile.Builders {
		builderTable := termtables.CreateTable()
		builderTable.AddRow("Name", builder.Name)
		builderTable.AddRow("Mount destination", builder.MountDestination)
		builderTable.AddRow("Dev command", strings.Join(builder.DevCommand, " "))

		for _, envKey := range builder.PassEnvs {
			setOrNot := "✗ (not set)"
			if os.Getenv(envKey) != "" {
				setOrNot = "✓ (set)"
			}

			builderTable.AddRow(fmt.Sprintf("ENV(%s)", envKey), setOrNot)
		}

		fmt.Printf("BUILDER\n%s\n", builderTable.Render())
	}

	for _, image := range bobfile.DockerImages {
		imageTable := termtables.CreateTable()
		imageTable.AddRow("Image", image.Image)
		imageTable.AddRow("Dockerfile path", image.DockerfilePath)

		fmt.Printf("DOCKER IMAGE\n%s\n", imageTable.Render())
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
