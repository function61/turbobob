package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

/*
func buildInCi(bobfile *Bobfile) error {
	revisionId := os.Getenv("CI_REVISION_ID")
	if revisionId == "" {
		return ErrCiRevisionIdEnvNotSet
	}

	metadata := revisionMetadataFromFull(revisionId, "managedByCi")

	return buildCommon(bobfile, metadata)
}
*/

func buildAndRunOneBuilder(builder BuilderSpec, bobfile *Bobfile, metadata *BuildMetadata, publishArtefacts bool) error {
	wd, errWd := os.Getwd()
	if errWd != nil {
		return errWd
	}

	imageName := builderImageName(bobfile, builder.Name)

	printHeading(fmt.Sprintf("Building builder %s (as %s)", builder.Name, imageName))

	if err := buildBuilder(bobfile, &builder); err != nil {
		return err
	}

	printHeading(fmt.Sprintf("Building with %s", builder.Name))

	buildArgs := []string{
		"docker",
		"run",
		"--rm",
		"--tty",
		"--volume", wd + ":" + builder.MountDestinationOrDefaultToApp(),
		"--volume", "/tmp/bob-tmp:/tmp",
	}

	// inserts ["--env", "FOO"] pairs for each PassEnvs
	buildArgs, errEnv := dockerRelayEnvVars(buildArgs, metadata, publishArtefacts, builder.PassEnvs)
	if errEnv != nil {
		return errEnv
	}

	buildArgs = append(buildArgs, imageName)

	buildCmd := passthroughStdoutAndStderr(exec.Command(buildArgs[0], buildArgs[1:]...))

	if err := buildCmd.Run(); err != nil {
		return err
	}

	return nil
}

func buildAndPushOneDockerImage(dockerImage DockerImageSpec, metadata *BuildMetadata, publishArtefacts bool) error {
	tagWithoutVersion := dockerImage.Image
	tag := tagWithoutVersion + ":" + metadata.FriendlyRevisionId
	dockerfilePath := dockerImage.DockerfilePath

	printHeading(fmt.Sprintf("Building %s", tag))

	buildCmd := passthroughStdoutAndStderr(exec.Command(
		"docker",
		"build",
		"--file", dockerfilePath,
		"--tag", tag,
		"."))

	if err := buildCmd.Run(); err != nil {
		return err
	}

	if publishArtefacts {
		printHeading(fmt.Sprintf("Pushing %s", tag))

		pushCmd := passthroughStdoutAndStderr(exec.Command(
			"docker",
			"push",
			tag))

		if err := pushCmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func buildCommon(bobfile *Bobfile, metadata *BuildMetadata, publishArtefacts bool) error {
	for _, builder := range bobfile.Builders {
		if err := buildAndRunOneBuilder(builder, bobfile, metadata, publishArtefacts); err != nil {
			return err
		}
	}

	if len(bobfile.DockerImages) > 0 {
		if err := loginToDockerHub(); err != nil {
			return err
		}

		for _, dockerImage := range bobfile.DockerImages {
			if err := buildAndPushOneDockerImage(dockerImage, metadata, publishArtefacts); err != nil {
				return err
			}
		}
	}

	return nil
}

func build(publishArtefacts bool) error {
	bobfile, errBobfile := readBobfile()
	if errBobfile != nil {
		return errBobfile
	}

	metadata, err := resolveMetadataFromVersionControl()
	if err != nil {
		return err
	}

	return buildCommon(bobfile, metadata, publishArtefacts)
}

func buildEntry() *cobra.Command {
	publishArtefacts := false

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds the project",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(build(publishArtefacts))
		},
	}

	cmd.Flags().BoolVarP(&publishArtefacts, "publish-artefacts", "p", publishArtefacts, "Whether to publish the artefacts")

	return cmd
}
