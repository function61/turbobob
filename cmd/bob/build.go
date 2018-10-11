package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

type BuildContext struct {
	Bobfile           *Bobfile
	OriginDir         string // where the repo exists
	WorkspaceDir      string // where the revision is being built
	PublishArtefacts  bool
	CloningStepNeeded bool // not done in CI
	VersionControl    Versioncontrol
	BuildMetadata     *BuildMetadata
}

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
		"--volume", wd + "/" + builder.MountSource + ":" + builder.MountDestination,
		"--volume", "/tmp/bob-tmp:/tmp",
	}

	// inserts ["--env", "FOO"] pairs for each PassEnvs
	buildArgs, errEnv := dockerRelayEnvVars(
		buildArgs,
		buildCtx.BuildMetadata,
		buildCtx.PublishArtefacts,
		builder.PassEnvs,
		true)
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

func cloneToWorkdir(buildCtx *BuildContext) error {
	rootForProject := projectSpecificDir(buildCtx.Bobfile.ProjectName, "")
	rootForProjectExists, rootForProjectExistsErr := fileExists(rootForProject)
	if rootForProjectExistsErr != nil {
		return rootForProjectExistsErr
	}

	if !rootForProjectExists {
		printHeading(fmt.Sprintf("Creating project root %s", rootForProject))

		if errMkdir := os.MkdirAll(rootForProject, 0700); errMkdir != nil {
			return errMkdir
		}
	}

	workspaceDirExists, workspaceDirExistsErr := fileExists(buildCtx.WorkspaceDir)
	if workspaceDirExistsErr != nil {
		return workspaceDirExistsErr
	}

	if !workspaceDirExists {
		printHeading(fmt.Sprintf("%s does not exist; cloning", buildCtx.WorkspaceDir))

		if err := buildCtx.VersionControl.CloneTo(buildCtx.WorkspaceDir); err != nil {
			return err
		}
	}

	workspaceRepo, errWorkspaceRepo := determineVcForDirectory(buildCtx.WorkspaceDir)
	if errWorkspaceRepo != nil {
		return errWorkspaceRepo
	}

	printHeading(fmt.Sprintf("Changing dir to %s", buildCtx.WorkspaceDir))

	if err := os.Chdir(buildCtx.WorkspaceDir); err != nil {
		return err
	}

	/*
		printHeading("Pulling")

		if err := workspaceRepo.Pull(); err != nil {
			return err
		}
	*/

	printHeading(fmt.Sprintf("Updating to %s", buildCtx.BuildMetadata.RevisionId))

	if err := workspaceRepo.Update(buildCtx.BuildMetadata.RevisionId); err != nil {
		return err
	}

	return nil
}

func buildCommon(buildCtx *BuildContext) error {
	if buildCtx.CloningStepNeeded {
		if err := cloneToWorkdir(buildCtx); err != nil {
			return err
		}
	}

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

func constructBuildContext(publishArtefacts bool, onlyCommitted bool) (*BuildContext, error) {
	bobfile, errBobfile := readBobfile()
	if errBobfile != nil {
		return nil, errBobfile
	}

	repoOriginDir, errGetwd := os.Getwd()
	if errGetwd != nil {
		return nil, errGetwd
	}

	versionControl, errVcDetermine := determineVcForDirectory(repoOriginDir)
	if errVcDetermine != nil {
		return nil, errVcDetermine
	}

	metadata, err := resolveMetadataFromVersionControl(versionControl, onlyCommitted)
	if err != nil {
		return nil, err
	}

	areWeInCi := os.Getenv("CI_REVISION_ID") != ""

	cloningStepNeeded := !areWeInCi && onlyCommitted

	buildCtx := &BuildContext{
		Bobfile:           bobfile,
		PublishArtefacts:  publishArtefacts,
		BuildMetadata:     metadata,
		OriginDir:         repoOriginDir,
		WorkspaceDir:      projectSpecificDir(bobfile.ProjectName, "workspace"),
		CloningStepNeeded: cloningStepNeeded,
		VersionControl:    versionControl,
	}

	return buildCtx, nil
}

func projectSpecificDir(projectName string, dirName string) string {
	return "/tmp/bob/" + projectName + "/" + dirName
}

func build(publishArtefacts bool, uncommitted bool) error {
	buildCtx, err := constructBuildContext(publishArtefacts, uncommitted)
	if err != nil {
		return err
	}

	return buildCommon(buildCtx)
}

func buildEntry() *cobra.Command {
	publishArtefacts := false
	uncommitted := false

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds the project",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(build(publishArtefacts, !uncommitted))
		},
	}

	cmd.Flags().BoolVarP(&publishArtefacts, "publish-artefacts", "p", publishArtefacts, "Whether to publish the artefacts")
	cmd.Flags().BoolVarP(&uncommitted, "uncommitted", "u", uncommitted, "Include uncommitted changes")

	return cmd
}
