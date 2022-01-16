package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/versioncontrol"
	"github.com/spf13/cobra"
)

func devCommand(builderName string, envsAreRequired bool, ignoreNag bool) ([]string, error) {
	bobfile, err := readBobfile()
	if err != nil {
		return nil, err
	}

	userConfig, err := loadUserconfigFile()
	if err != nil {
		return nil, err
	}

	// this is a natural point to check for repository's quality warnings. these are not issues
	// that should break the build, but are severe enough to bug a maintainer
	if !ignoreNag {
		if err := qualityCheckBuilderUsesExpect(userConfig.ProjectQuality.BuilderUsesExpect, bobfile); err != nil {
			return nil, err
		}

		if err := qualityCheckFiles(userConfig.ProjectQuality.FileRules); err != nil {
			return nil, err
		}
	}

	wd, errWd := os.Getwd()
	if errWd != nil {
		return nil, errWd
	}

	builder, err := findBuilder(bobfile, builderName)
	if err != nil {
		return nil, err
	}

	for _, subrepo := range bobfile.Subrepos {
		if err := ensureSubrepoCloned(wd+"/"+subrepo.Destination, subrepo); err != nil {
			return nil, err
		}
	}

	// defaults to false now because it's still a bit buggy
	enablePromptCustomization := userConfig.EnablePromptCustomization != nil && *userConfig.EnablePromptCustomization

	shimCfg := shimConfig{
		BuilderName:               builder.Name,
		EnablePromptCustomization: enablePromptCustomization,
		DynamicProTipsFromHost:    []string{},
	}

	if len(builder.DevPorts) > 0 {
		shimCfg.DynamicProTipsFromHost = append(shimCfg.DynamicProTipsFromHost, fmt.Sprintf(
			"mapped dev ports: %s",
			strings.Join(builder.DevPorts, ", ")))
	}

	containerName := devContainerName(bobfile, *builder)

	useShim := true // TODO: always use?

	var dockerCmd []string
	if isDevContainerRunning(containerName) {
		dockerCmd = []string{
			"docker",
			"exec",
			"--interactive",
			"--tty"}

		if builder.Workdir != "" {
			dockerCmd = append(dockerCmd, "--workdir", builder.Workdir)
		}

		dockerCmd = append(dockerCmd, containerName)

		if useShim {
			dockerCmd = append(dockerCmd, "bob", "dev-shim", "--")
		}

		dockerCmd = append(dockerCmd, builder.Commands.Dev...)
	} else {
		builderType, _, err := parseBuilderUsesType(builder.Uses)
		if err != nil {
			return nil, err
		}

		// only need to build if a builder is dockerfile. images are ready for consumption.
		if builderType == builderUsesTypeDockerfile {
			// internally prints heading
			if err := buildBuilder(bobfile, builder); err != nil {
				return nil, err
			}
		}

		dockerCmd = []string{
			"docker",
			"run",
			"--rm",
			"--interactive",
			"--tty",
			"--user", fmt.Sprintf("0:%d", os.Getgid()), // root user for now, but use user's group so we have a chance at having sane group permissions
			"--name", containerName,
			"--entrypoint=", // turn off possible "arg mode" in base image (our cmd would just be args to entrypoint)
			"--volume", wd + "/" + builder.MountSource + ":" + builder.MountDestination,
			"--volume", "/tmp/build:/tmp/build", // cannot map to /tmp because at least apt won't work (permission issues?)
		}

		fixInputRc := true
		if fixInputRc {
			// https://superuser.com/a/589629
			dockerCmd = append(dockerCmd, "--volume", "/etc/inputrc:/etc/inputrc:ro")
		}

		if builder.Workdir != "" {
			dockerCmd = append(dockerCmd, "--workdir", builder.Workdir)
		}

		for _, port := range builder.DevPorts {
			dockerCmd = append(dockerCmd, "--publish", port)
		}

		devHttpIngress, ingressHostname := setupDevIngress(
			builder,
			userConfig.DevIngressSettings,
			bobfile)
		if len(devHttpIngress) > 0 {
			dockerCmd = append(dockerCmd, devHttpIngress...)

			shimCfg.DynamicProTipsFromHost = append(shimCfg.DynamicProTipsFromHost, fmt.Sprintf("dev ingress: https://%s/", ingressHostname))
		}

		archesToBuildFor := buildArchOnlyForCurrentlyRunningArch(*bobfile.OsArches)

		// inserts ["--env", "FOO"] pairs for each PassEnvs
		var errEnv error
		dockerCmd, errEnv = dockerRelayEnvVars(
			dockerCmd,
			revisionIdForDev(),
			*builder,
			envsAreRequired,
			archesToBuildFor,
			false)
		if errEnv != nil {
			return nil, errEnv
		}

		if useShim {
			ourPath, err := os.Executable()
			if err != nil {
				return nil, err
			}

			// this needs to be dynamic, because on the host side there must be a unique dir
			// per dev container
			shimDataDirHost, err := ioutil.TempDir("", "bob-shim-")
			if err != nil {
				return nil, err
			}

			if err := jsonfile.Write(filepath.Join(shimDataDirHost, shimConfigFile), &shimCfg); err != nil {
				return nil, err
			}

			dockerCmd = append(dockerCmd, "--volume", shimDataDirHost+":"+shimDataDirContainer+":ro")

			// mount this executable inside the container, so we'll be able to use the shim
			dockerCmd = append(dockerCmd, "--volume", ourPath+":/bin/bob:ro")
		}

		dockerCmd = append(dockerCmd, builderImageName(bobfile, *builder))

		if useShim {
			// inject a shim to start the shell indirectly, so we can do preparations like:
			// - inject commands into history
			// - set up build cache paths
			// - show pro-tips
			dockerCmd = append(dockerCmd, "bob", "dev-shim", "--")
		}

		dockerCmd = append(dockerCmd, builder.Commands.Dev...)
	}

	return dockerCmd, nil
}

func enterInteractiveDevContainer(dockerCmd []string) error {
	//nolint:gosec // ok
	cmd := exec.Command(dockerCmd[0], dockerCmd[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func devEntry() *cobra.Command {
	norequireEnvs := false
	dry := false
	ignoreNag := false

	cmd := &cobra.Command{
		Use:   "dev [builderName]",
		Short: "Enter builder container in dev mode",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			builderName := "default"
			if len(args) >= 1 {
				builderName = args[0]
			}

			osutil.ExitIfError(func() error {
				dockerCommand, err := devCommand(builderName, !norequireEnvs, ignoreNag)
				if err != nil {
					return err
				}

				if !dry {
					return enterInteractiveDevContainer(dockerCommand)
				} else {
					_, err = fmt.Println(strings.Join(dockerCommand, " "))
					return err
				}
			}())
		},
	}

	cmd.Flags().BoolVarP(&norequireEnvs, "norequire-envs", "n", norequireEnvs, "Don´t error out if not all ENV vars are set")
	cmd.Flags().BoolVarP(&dry, "dry", "", dry, "Just print out the dev command (you may need to do something exotic)")
	cmd.Flags().BoolVarP(&ignoreNag, "ignore-nag", "", ignoreNag, "Ignore project quality warning nags")

	return cmd
}

func currentRunningGoOsArchToOsArchCode() OsArchCode {
	return OsArchCode(fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH))
}

// TODO: maybe merge with resolveMetadataFromVersionControl(.., false)
func revisionIdForDev() *versioncontrol.RevisionId {
	return &versioncontrol.RevisionId{
		VcKind:             "managedByCi", // FIXME
		RevisionId:         "dev",
		RevisionIdShort:    "dev",
		FriendlyRevisionId: "dev",
	}
}

func buildArchOnlyForCurrentlyRunningArch(archesToBuildFor OsArchesSpec) OsArchesSpec {
	devMachineArch := osArchCodeToOsArchesSpec(currentRunningGoOsArchToOsArchCode())

	// to speed up dev, build only for the arch we're running now, but only if arches
	// intersect (if project wants to build only for "neutral", and we're running on
	// "linux-amd64", we wouldn't want to ask to build for "linux-amd64" since it's unsupported)
	if osArchesIntersects(archesToBuildFor, devMachineArch) {
		return devMachineArch
	}

	return archesToBuildFor
}
