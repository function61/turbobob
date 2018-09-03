package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

type BuildContext struct {
	Bobfile          *Bobfile
	PublishArtefacts bool
	BuildMetadata    *BuildMetadata
}

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

func buildAndRunOneBuilder(builder BuilderSpec, buildCtx *BuildContext) error {
	wd, errWd := os.Getwd()
	if errWd != nil {
		return errWd
	}

	imageName := builderImageName(buildCtx.Bobfile, builder.Name)

	printHeading(fmt.Sprintf("Building builder %s (as %s)", builder.Name, imageName))

	if err := buildBuilder(buildCtx.Bobfile, &builder); err != nil {
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
	buildArgs, errEnv := dockerRelayEnvVars(buildArgs, buildCtx.BuildMetadata, buildCtx.PublishArtefacts, builder.PassEnvs)
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

func buildAndPushOneDockerImage(dockerImage DockerImageSpec, buildCtx *BuildContext) error {
	tagWithoutVersion := dockerImage.Image
	tag := tagWithoutVersion + ":" + buildCtx.BuildMetadata.FriendlyRevisionId
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

	if buildCtx.PublishArtefacts {
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

func buildCommon(buildCtx *BuildContext) error {
	for _, builder := range buildCtx.Bobfile.Builders {
		if err := buildAndRunOneBuilder(builder, buildCtx); err != nil {
			return err
		}
	}

	for _, dockerImage := range buildCtx.Bobfile.DockerImages {
		if buildCtx.PublishArtefacts {
			if err := loginToDockerRegistry(dockerImage); err != nil {
				return err
			}
		}

		if err := buildAndPushOneDockerImage(dockerImage, buildCtx); err != nil {
			return err
		}
	}

	return nil
}

func constructBuildContext(publishArtefacts bool) (*BuildContext, error) {
	bobfile, errBobfile := readBobfile()
	if errBobfile != nil {
		return nil, errBobfile
	}

	metadata, err := resolveMetadataFromVersionControl()
	if err != nil {
		return nil, err
	}

	buildCtx := &BuildContext{
		Bobfile:          bobfile,
		PublishArtefacts: publishArtefacts,
		BuildMetadata:    metadata,
	}

	return buildCtx, nil
}

func build(publishArtefacts bool) error {
	buildCtx, err := constructBuildContext(publishArtefacts)
	if err != nil {
		return err
	}

	return buildCommon(buildCtx)
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
