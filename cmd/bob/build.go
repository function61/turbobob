package main

import (
	"fmt"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/turbobob/pkg/versioncontrol"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

type BuildContext struct {
	Bobfile           *Bobfile
	OriginDir         string // where the repo exists
	WorkspaceDir      string // where the revision is being built
	PublishArtefacts  bool
	CloningStepNeeded bool // false in CI, true in local (unless uncommited build requested)
	BuilderNameFilter string
	ENVsAreRequired   bool
	VersionControl    versioncontrol.Interface
	RevisionId        *versioncontrol.RevisionId
	FastBuild         bool // skip all non-essential steps (linting, testing etc.) to build faster
}

func runBuilder(builder BuilderSpec, buildCtx *BuildContext, opDesc string, cmdToRun []string) error {
	wd, errWd := os.Getwd()
	if errWd != nil {
		return errWd
	}

	printHeading(fmt.Sprintf("%s/%s (%s)", builder.Name, opDesc, builderCommandToHumanReadable(cmdToRun)))

	buildArgs := []string{
		"docker",
		"run",
		"--rm",
		"--tty",
		"--volume", wd + "/" + builder.MountSource + ":" + builder.MountDestination,
		"--volume", "/tmp/build:/tmp/build", // cannot map to /tmp because at least apt won't work (permission issues?)
	}

	if builder.Workdir != "" {
		buildArgs = append(buildArgs, "--workdir", builder.Workdir)
	}

	archesToBuildFor := *buildCtx.Bobfile.OsArches

	if buildCtx.FastBuild {
		// skip building other arches than what we're running on now
		archesToBuildFor = buildArchOnlyForCurrentlyRunningArch(archesToBuildFor)
	}

	// inserts ["--env", "FOO"] pairs for each PassEnvs
	buildArgs, errEnv := dockerRelayEnvVars(
		buildArgs,
		buildCtx.RevisionId,
		buildCtx.PublishArtefacts,
		builder,
		buildCtx.ENVsAreRequired,
		archesToBuildFor,
		buildCtx.FastBuild)
	if errEnv != nil {
		return errEnv
	}

	buildArgs = append(buildArgs, builderImageName(buildCtx.Bobfile, builder))

	if len(cmdToRun) > 0 {
		buildArgs = append(buildArgs, cmdToRun...)
	}

	buildCmd := passthroughStdoutAndStderr(exec.Command(buildArgs[0], buildArgs[1:]...))

	if err := buildCmd.Run(); err != nil {
		return err
	}

	return nil
}

func buildAndPushOneDockerImage(dockerImage DockerImageSpec, buildCtx *BuildContext) error {
	tagWithoutVersion := dockerImage.Image
	tag := tagWithoutVersion + ":" + buildCtx.RevisionId.FriendlyRevisionId
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
	rootForProjectExists, rootForProjectExistsErr := fileexists.Exists(rootForProject)
	if rootForProjectExistsErr != nil {
		return rootForProjectExistsErr
	}

	if !rootForProjectExists {
		printHeading(fmt.Sprintf("Creating project root %s", rootForProject))

		if errMkdir := os.MkdirAll(rootForProject, 0700); errMkdir != nil {
			return errMkdir
		}
	}

	workspaceDirExists, workspaceDirExistsErr := fileexists.Exists(buildCtx.WorkspaceDir)
	if workspaceDirExistsErr != nil {
		return workspaceDirExistsErr
	}

	if !workspaceDirExists {
		printHeading(fmt.Sprintf("%s does not exist; cloning", buildCtx.WorkspaceDir))

		cloned := buildCtx.VersionControl.WithAnotherDir(buildCtx.WorkspaceDir)
		if err := cloned.CloneFrom(buildCtx.OriginDir); err != nil {
			return err
		}
	}

	workspaceRepo, errWorkspaceRepo := versioncontrol.DetectForDirectory(buildCtx.WorkspaceDir)
	if errWorkspaceRepo != nil {
		return errWorkspaceRepo
	}

	printHeading(fmt.Sprintf("Changing dir to %s", buildCtx.WorkspaceDir))

	if err := os.Chdir(buildCtx.WorkspaceDir); err != nil {
		return err
	}

	printHeading("Pulling")

	if err := workspaceRepo.Pull(); err != nil {
		return err
	}

	printHeading(fmt.Sprintf("Updating to %s", buildCtx.RevisionId.RevisionId))

	if err := workspaceRepo.Update(buildCtx.RevisionId.RevisionId); err != nil {
		return err
	}

	return nil
}

func build(buildCtx *BuildContext) error {
	if buildCtx.CloningStepNeeded {
		if err := cloneToWorkdir(buildCtx); err != nil {
			return err
		}
	}

	for _, subrepo := range buildCtx.Bobfile.Subrepos {
		if err := ensureSubrepoCloned(buildCtx.WorkspaceDir+"/"+subrepo.Destination, subrepo); err != nil {
			return err
		}
	}

	// build builders (TODO: check cache so this is not done unless necessary?)
	for _, builder := range buildCtx.Bobfile.Builders {
		builder := builder // pin

		if buildCtx.BuilderNameFilter != "" && builder.Name != buildCtx.BuilderNameFilter {
			continue
		}

		builderType, _, err := parseBuilderUsesType(builder.Uses)
		if err != nil {
			return err
		}

		// only need to build if a builder is dockerfile. images are ready for consumption.
		if builderType != builderUsesTypeDockerfile {
			continue
		}

		if err := buildBuilder(buildCtx.Bobfile, &builder); err != nil {
			return err
		}
	}

	// two passes:
	//   1) run all builds with all phases except publish
	//   2) run all builds with only publish phase (but only if publish requested)

	// 1)
	for _, builder := range buildCtx.Bobfile.Builders {
		if buildCtx.BuilderNameFilter != "" && builder.Name != buildCtx.BuilderNameFilter {
			continue
		}

		if err := runBuilder(builder, buildCtx, "build", builder.Commands.Build); err != nil {
			return err
		}
	}

	// 2)
	for _, builder := range buildCtx.Bobfile.Builders {
		if buildCtx.BuilderNameFilter != "" && builder.Name != buildCtx.BuilderNameFilter {
			continue
		}

		if !buildCtx.PublishArtefacts || len(builder.Commands.Publish) == 0 {
			continue
		}

		if err := runBuilder(builder, buildCtx, "publish", builder.Commands.Publish); err != nil {
			return err
		}
	}

	for _, dockerImage := range buildCtx.Bobfile.DockerImages {
		if buildCtx.PublishArtefacts {
			// FIXME: login to each different registry only once
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

func constructBuildContext(
	publishArtefacts bool,
	onlyCommitted bool,
	builderNameFilter string,
	envsAreRequired bool,
	fastBuild bool,
) (*BuildContext, error) {
	bobfile, errBobfile := readBobfile()
	if errBobfile != nil {
		return nil, errBobfile
	}

	repoOriginDir, errGetwd := os.Getwd()
	if errGetwd != nil {
		return nil, errGetwd
	}

	versionControl, errVcDetermine := versioncontrol.DetectForDirectory(repoOriginDir)
	if errVcDetermine != nil {
		return nil, errVcDetermine
	}

	metadata, err := versioncontrol.CurrentRevisionId(versionControl, onlyCommitted)
	if err != nil {
		return nil, err
	}

	areWeInCi := os.Getenv("CI_REVISION_ID") != ""

	workspaceDir := projectSpecificDir(bobfile.ProjectName, "workspace")

	cloningStepNeeded := !areWeInCi && onlyCommitted

	if !cloningStepNeeded {
		workspaceDir = repoOriginDir
	}

	buildCtx := &BuildContext{
		Bobfile:           bobfile,
		PublishArtefacts:  publishArtefacts,
		RevisionId:        metadata,
		OriginDir:         repoOriginDir,
		WorkspaceDir:      workspaceDir,
		CloningStepNeeded: cloningStepNeeded,
		BuilderNameFilter: builderNameFilter,
		ENVsAreRequired:   envsAreRequired,
		VersionControl:    versionControl,
		FastBuild:         fastBuild,
	}

	return buildCtx, nil
}

func projectSpecificDir(projectName string, dirName string) string {
	return "/tmp/bob/" + projectName + "/" + dirName
}

func buildEntry() *cobra.Command {
	publishArtefacts := false
	uncommitted := false
	builderName := ""
	norequireEnvs := false
	fastbuild := false

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds the project",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			buildCtx, err := constructBuildContext(publishArtefacts, !uncommitted, builderName, !norequireEnvs, fastbuild)
			reactToError(err)
			if err := build(buildCtx); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVarP(&norequireEnvs, "norequire-envs", "n", norequireEnvs, "Don´t error out if not all ENV vars are set")
	cmd.Flags().BoolVarP(&publishArtefacts, "publish-artefacts", "p", publishArtefacts, "Whether to publish the artefacts")
	cmd.Flags().BoolVarP(&uncommitted, "uncommitted", "u", uncommitted, "Include uncommitted changes")
	cmd.Flags().BoolVarP(&fastbuild, "fast", "f", fastbuild, "Skip non-essential steps (linting, testing etc.)")
	cmd.Flags().StringVarP(&builderName, "builder", "b", builderName, "If specified, runs only one builder instead of all")

	return cmd
}
