package main

import (
	"fmt"
	"strings"

	"github.com/function61/gokit/os/osutil"
	"github.com/scylladb/termtables"
	"github.com/spf13/cobra"
)

func info() error {
	// FIXME: too many assumptions
	buildCtx, err := constructBuildContext(true, true, "", true, false, false)
	if err != nil {
		return err
	}

	revisionId := buildCtx.RevisionId // shorthand

	basicDetails := termtables.CreateTable()
	basicDetails.AddRow("Project name", buildCtx.Bobfile.ProjectName)
	basicDetails.AddRow("VcKind", revisionId.VcKind)
	basicDetails.AddRow("Revision ID (full)", fmt.Sprintf("%s (%s)", revisionId.RevisionIdShort, revisionId.RevisionId))
	basicDetails.AddRow("Friendly revision", revisionId.FriendlyRevisionId)

	fmt.Printf("BASIC DETAILS\n%s\n", basicDetails.Render())

	checkMarkSetNotSet := boolToStringTheme{"✓ (set)", "✗ (not set)"}

	for _, builder := range buildCtx.Bobfile.Builders {
		ports := "(none)"

		if len(builder.DevPorts) > 0 {
			ports = strings.Join(builder.DevPorts, ", ")
		}

		builderTable := termtables.CreateTable()
		builderTable.AddRow("Name", builder.Name)
		builderTable.AddRow("Uses", builder.Uses)
		builderTable.AddRow("Mount source", builder.MountSource)
		builderTable.AddRow("Mount destination", builder.MountDestination)
		builderTable.AddRow("Build command", builderCommandToHumanReadable(builder.Commands.Build))
		builderTable.AddRow("Publish command", strings.Join(builder.Commands.Publish, " "))
		builderTable.AddRow("Dev command", strings.Join(builder.Commands.Dev, " "))
		builderTable.AddRow("Dev ports", ports)

		for _, envKey := range builder.PassEnvs {
			builderTable.AddRow(fmt.Sprintf("ENV(%s)", envKey), checkMarkSetNotSet.String(isEnvVarPresent(envKey)))
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
		checksTable.AddRow(check.Name, checkMark.String(check.Ok), check.Reason)
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
			osutil.ExitIfError(info())
		},
	}
}

func builderCommandToHumanReadable(cmd []string) string {
	cmdHumanReadable := strings.Join(cmd, " ")
	if cmdHumanReadable == "" {
		cmdHumanReadable = "(default command of Docker image)"
	}
	return cmdHumanReadable
}

var checkMark = boolToStringTheme{"✓", "✗"}

type boolToStringTheme struct {
	positive string
	negative string
}

func (c boolToStringTheme) String(positive bool) string {
	if positive {
		return c.positive
	} else {
		return c.negative
	}
}
