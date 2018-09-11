package main

import (
	"fmt"
	"github.com/apcera/termtables"
	"github.com/spf13/cobra"
	"strings"
)

func info() error {
	buildCtx, err := constructBuildContext(true) // FIXME: static publishArtefacts
	if err != nil {
		return err
	}

	metadata := buildCtx.BuildMetadata

	basicDetails := termtables.CreateTable()
	basicDetails.AddRow("Project name", buildCtx.Bobfile.ProjectName)
	basicDetails.AddRow("VcKind", metadata.VcKind)
	basicDetails.AddRow("Revision ID (full)", fmt.Sprintf("%s (%s)", metadata.RevisionIdShort, metadata.RevisionId))
	basicDetails.AddRow("Friendly revision", metadata.FriendlyRevisionId)

	fmt.Printf("BASIC DETAILS\n%s\n", basicDetails.Render())

	for _, builder := range buildCtx.Bobfile.Builders {
		ports := "(none)"

		if len(builder.DevPorts) > 0 {
			ports = strings.Join(builder.DevPorts, ", ")
		}

		builderTable := termtables.CreateTable()
		builderTable.AddRow("Name", builder.Name)
		builderTable.AddRow("Mount destination", builder.MountDestination)
		builderTable.AddRow("Dev command", strings.Join(builder.DevCommand, " "))
		builderTable.AddRow("Dev ports", ports)

		for _, envKey := range builder.PassEnvs {
			setOrNot := "✗ (not set)"
			if isEnvVarPresent(envKey) {
				setOrNot = "✓ (set)"
			}

			builderTable.AddRow(fmt.Sprintf("ENV(%s)", envKey), setOrNot)
		}

		fmt.Printf("BUILDER\n%s\n", builderTable.Render())
	}

	for _, image := range buildCtx.Bobfile.DockerImages {
		imageTable := termtables.CreateTable()
		imageTable.AddRow("Image", image.Image)
		imageTable.AddRow("Dockerfile path", image.DockerfilePath)

		fmt.Printf("DOCKER IMAGE\n%s\n", imageTable.Render())
	}

	checksTable := termtables.CreateTable()
	checksTable.AddHeaders("CHECKS", "Ok", "Reason")

	checksResults, errRunningChecks := RunChecks(buildCtx)
	if errRunningChecks != nil {
		return errRunningChecks
	}

	for _, check := range checksResults {
		okChar := "✓"
		if !check.Ok {
			okChar = "✗"
		}

		checksTable.AddRow(check.Name, okChar, check.Reason)
	}

	fmt.Printf("%s\n", checksTable.Render())

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
