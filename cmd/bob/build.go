package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/versioncontrol"
	"github.com/spf13/cobra"
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
	Debug             bool   // enables additional debugging or verbose logging
	FastBuild         bool   // skip all non-essential steps (linting, testing etc.) to build faster
	RepositoryURL     string // human-visitable URL, like "https://github.com/function61/turbobob"
}

func runBuilder(builder BuilderSpec, buildCtx *BuildContext, opDesc string, cmdToRun []string) error {
	wd, errWd := os.Getwd()
	if errWd != nil {
		return errWd
	}

	builderNameOpDesc := fmt.Sprintf("%s/%s", builder.Name, opDesc)

	switch must(parseBuilderUsesType(builder.Uses)) {
	case builderUsesTypeImage:
		// the "$ docker run ..." later would do an implicit pull, but let's do an explicit pull
		// here in order to nicely put the download progress output in its own log line group
		if err := withLogLineGroup(fmt.Sprintf("%s > pull", builderNameOpDesc), func() error {
			return dockerPullIfRequired(builderImageName(buildCtx.Bobfile, builder))
		}); err != nil {
			return err
		}
	case builderUsesTypeDockerfile:
		// no-op (doesn't need a pull)
	default:
		panic("unknown builderType")
	}

	// empty work just to emit a "starting" log group. this log group is important because if the
	// command-to-run itself doesn't create log groups (to which we could insert script name), then
	// script name won't be visible at all in none of the group names
	_ = withLogLineGroup(fmt.Sprintf("%s > starting %s", builderNameOpDesc, builderCommandToHumanReadable(cmdToRun)), func() error { return nil })

	buildArgs := []string{
		"docker",
		"run",
		"--rm",
		"--tty",
		"--entrypoint=", // turn off possible "arg mode" in base image (our cmd would just be args to entrypoint)
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
		builder,
		buildCtx.ENVsAreRequired,
		archesToBuildFor,
		buildCtx.FastBuild,
		buildCtx.Debug)
	if errEnv != nil {
		return errEnv
	}

	buildArgs = append(buildArgs, builderImageName(buildCtx.Bobfile, builder))

	if len(cmdToRun) > 0 {
		buildArgs = append(buildArgs, cmdToRun...)
	}

	//nolint:gosec // ok
	buildCmd := exec.Command(buildArgs[0], buildArgs[1:]...)
	buildCmd.Stdout = newLineSplitterTee(io.Discard, func(line string) {
		lineMaybeModified := func() string {
			// for each log line group, add "breadcrumb prefix" of builder / operation description.
			// example group name: "staticAnalysis" => "default/build > staticAnalysis"
			if strings.HasPrefix(line, "::group::") {
				originalGroupName := line[len("::group::"):]

				return fmt.Sprintf("::group::%s > %s", builderNameOpDesc, originalGroupName)
			} else {
				return line // as-is
			}
		}()

		_, _ = os.Stdout.Write([]byte(lineMaybeModified + "\n")) // need to add newline back
	})
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return err
	}

	return nil
}

func buildAndPushOneDockerImage(dockerImage DockerImageSpec, buildCtx *BuildContext) error {
	tagWithoutVersion := dockerImage.Image
	tag := tagWithoutVersion + ":" + buildCtx.RevisionId.FriendlyRevisionId
	tagLatest := tagWithoutVersion + ":latest"
	dockerfilePath := dockerImage.DockerfilePath

	labelArgs := []string{
		"--label=org.opencontainers.image.created=" + time.Now().UTC().Format(time.RFC3339),
		"--label=org.opencontainers.image.revision=" + buildCtx.RevisionId.RevisionId,
		"--label=org.opencontainers.image.version=" + buildCtx.RevisionId.FriendlyRevisionId,
	}

	if buildCtx.RepositoryURL != "" {
		// "URL to get source code for building the image"
		labelArgs = append(labelArgs, "--label=org.opencontainers.image.source="+buildCtx.RepositoryURL)
		// "URL to find more information on the image"
		labelArgs = append(labelArgs, "--label=org.opencontainers.image.url="+buildCtx.RepositoryURL)
	}

	// "" => "."
	// "Dockerfile" => "."
	// "subdir/Dockerfile" => "subdir"
	buildContextDir := filepath.Dir(dockerfilePath)

	printHeading(fmt.Sprintf("Building %s", tag))

	// use buildx when platforms set. it's almost same as "$ docker build" but it almost transparently
	// supports cross-architecture builds via binftm_misc + QEMU userspace emulation
	if len(dockerImage.Platforms) > 0 {
		// TODO: if in CI, install buildx automatically if needed?

		args := []string{
			"buildx",
			"build",
			"--platform", strings.Join(dockerImage.Platforms, ","),
			"--file", dockerfilePath,
			"--tag=" + tag,
		}

		args = append(args, labelArgs...)

		if dockerImage.TagLatest {
			args = append(args, "--tag="+tagLatest)
		}

		args = append(args, buildContextDir)

		if buildCtx.PublishArtefacts {
			// the build command has integrated push support. we'd actually prefer to separate
			// these stages, but multi-arch manifests aren't supported storing locally so we've
			// to push immediately
			args = append(args, "--push")
		}

		return passthroughStdoutAndStderr(exec.Command("docker", args...)).Run()
	}

	dockerBuildArgs := []string{"docker",
		"build",
		"--file", dockerfilePath,
		"--tag", tag}
	dockerBuildArgs = append(dockerBuildArgs, labelArgs...)
	dockerBuildArgs = append(dockerBuildArgs, buildContextDir)

	//nolint:gosec // ok
	buildCmd := passthroughStdoutAndStderr(exec.Command(dockerBuildArgs[0], dockerBuildArgs[1:]...))

	if err := buildCmd.Run(); err != nil {
		return err
	}

	if buildCtx.PublishArtefacts {
		pushTag := func(tag string) error {
			printHeading(fmt.Sprintf("Pushing %s", tag))

			pushCmd := passthroughStdoutAndStderr(exec.Command(
				"docker",
				"push",
				tag))

			if err := pushCmd.Run(); err != nil {
				return err
			}

			return nil
		}

		if err := pushTag(tag); err != nil {
			return err
		}

		if dockerImage.TagLatest {
			if err := exec.Command("docker", "tag", tag, tagLatest).Run(); err != nil {
				return fmt.Errorf("tagging failed %s -> %s failed: %v", tag, tagLatest, err)
			}

			if err := pushTag(tagLatest); err != nil {
				return err
			}
		}

		return nil
	}

	return nil
}

func cloneToWorkdir(buildCtx *BuildContext) error {
	rootForProject := projectSpecificDir(buildCtx.Bobfile.ProjectName, "")
	rootForProjectExists, rootForProjectExistsErr := osutil.Exists(rootForProject)
	if rootForProjectExistsErr != nil {
		return rootForProjectExistsErr
	}

	if !rootForProjectExists {
		printHeading(fmt.Sprintf("Creating project root %s", rootForProject))

		if errMkdir := os.MkdirAll(rootForProject, 0700); errMkdir != nil {
			return errMkdir
		}
	}

	workspaceDirExists, workspaceDirExistsErr := osutil.Exists(buildCtx.WorkspaceDir)
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

	// build builders.
	//
	// it would be cool to not invoke Docker if this is cached anyway, but the analysis would have
	// to include modification check for the Dockerfile and all of its build context, so we're just
	// best off calling Docker build because it is the best at detecting cache invalidation.
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

	// three-pass process. the flow is well documented in *BuilderCommands* type
	pass := func(opDesc string, getCommand func(cmds BuilderCommands) []string) error {
		for _, builder := range buildCtx.Bobfile.Builders {
			if buildCtx.BuilderNameFilter != "" && builder.Name != buildCtx.BuilderNameFilter {
				continue
			}

			cmd := getCommand(builder.Commands)
			if len(cmd) == 0 { // no command for this step specified
				continue
			}

			if err := runBuilder(builder, buildCtx, opDesc, cmd); err != nil {
				return fmt.Errorf("%s.%s: %w", builder.Name, opDesc, err)
			}
		}

		return nil
	}

	preparePass := func(cmds BuilderCommands) []string { return cmds.Prepare }
	buildPass := func(cmds BuilderCommands) []string { return cmds.Build }
	publishPass := func(cmds BuilderCommands) []string { return cmds.Publish }

	if err := pass("prepare", preparePass); err != nil {
		return err // err context ok
	}

	if err := pass("build", buildPass); err != nil {
		return err // err context ok
	}

	if err := pass("publish", publishPass); err != nil {
		return err // err context ok
	}

	dockerLoginCache := newDockerRegistryLoginCache()

	for _, dockerImage := range buildCtx.Bobfile.DockerImages {
		if buildCtx.BuilderNameFilter != "" {
			continue // when building a specifified builder => skip everything else
		}

		if buildCtx.PublishArtefacts {
			if err := loginToDockerRegistry(dockerImage, dockerLoginCache); err != nil {
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
	areWeInCi bool,
) (*BuildContext, error) {
	bobfile, err := readBobfile()
	if err != nil {
		return nil, err
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
			areWeInCi := os.Getenv("CI_REVISION_ID") != ""

			buildCtx, err := constructBuildContext(
				publishArtefacts,
				!uncommitted,
				builderName,
				!norequireEnvs,
				fastbuild,
				areWeInCi)
			osutil.ExitIfError(err)

			osutil.ExitIfError(build(buildCtx))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "in-ci-autodetect-settings",
		Short: "Run build in CI, autodetect build info (like if building for a pull request) from its ENV variables",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(func() error {
				if !runningInGitHubActions() {
					return errors.New("expecting GITHUB_ACTIONS=true")
				}

				publishArtefacts, err := func() (bool, error) {
					// https://docs.github.com/en/code-security/dependabot/working-with-dependabot/automating-dependabot-with-github-actions#responding-to-events
					isDependabot := os.Getenv("GITHUB_ACTOR") == "dependabot[bot]"

					event := os.Getenv("GITHUB_EVENT_NAME")
					switch event {
					case "push":
						if isDependabot { // one would imagine these would be PR events, but dependabot can cause pushes too ..
							// .. and as such shouldn't cause artefacts to be published (secrets aren't even accessible,
							// and publishing often requires secrets)
							return false, nil
						}

						return true, nil
					case "pull_request": // PRs don't publish artefacts
						return false, nil
					default:
						return false, fmt.Errorf("unsupported event: %s", event)
					}
				}()
				if err != nil {
					return err
				}

				buildCtx, err := constructBuildContext(publishArtefacts, true, "", false, false, true)
				if err != nil {
					return err
				}

				if ownerAndRepo := os.Getenv("GITHUB_REPOSITORY"); ownerAndRepo != "" {
					// "function61/turbobob" => "https://github.com/function61/turbobob"
					buildCtx.RepositoryURL = fmt.Sprintf("%s/%s", os.Getenv("GITHUB_SERVER_URL"), ownerAndRepo)
				}

				if os.Getenv("RUNNER_DEBUG") == "1" {
					buildCtx.Debug = true
				}

				return build(buildCtx)
			}())
		},
	})
	cmd.Flags().BoolVarP(&norequireEnvs, "norequire-envs", "n", norequireEnvs, "Don´t error out if not all ENV vars are set")
	cmd.Flags().BoolVarP(&publishArtefacts, "publish-artefacts", "p", publishArtefacts, "Whether to publish the artefacts")
	cmd.Flags().BoolVarP(&uncommitted, "uncommitted", "u", uncommitted, "Include uncommitted changes")
	cmd.Flags().BoolVarP(&fastbuild, "fast", "f", fastbuild, "Skip non-essential steps (linting, testing etc.)")
	cmd.Flags().StringVarP(&builderName, "builder", "b", builderName, "If specified, runs only one builder instead of all")

	return cmd
}

// same as build, but for running inside the container. only focuses on building artefacts -
// cannot publish, use version control etc.
func buildInsideEntry() *cobra.Command {
	fastBuild := false

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds the artefacts",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(buildInside(fastBuild))
		},
	}

	cmd.Flags().BoolVarP(&fastBuild, "fast", "f", fastBuild, "Skip non-essential steps (linting, testing etc.)")

	return cmd
}

func buildInside(fastBuild bool) error {
	bobfile, err := readBobfile()
	if err != nil {
		return err
	}

	shimCfg, err := readShimConfig()
	if err != nil {
		return err
	}

	builder, err := findBuilder(bobfile, shimCfg.BuilderName)
	if err != nil {
		return err
	}

	buildArgs := builder.Commands.Build

	//nolint:gosec // ok
	buildCmd := passthroughStdoutAndStderr(exec.Command(buildArgs[0], buildArgs[1:]...))

	// pass almost all of our environment variables
	for _, envSerialized := range os.Environ() {
		if key, _ := osutil.ParseEnv(envSerialized); key == "FASTBUILD" {
			// since we have explicit interface from here on, ignore setting any previous
			// value so we can control whether we set this or not
			continue
		}

		buildCmd.Env = append(buildCmd.Env, envSerialized)
	}

	if fastBuild {
		buildCmd.Env = append(buildCmd.Env, "FASTBUILD=true")
	}

	return buildCmd.Run()
}
