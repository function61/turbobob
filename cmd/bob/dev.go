package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

func devCommand(builderName string, envsAreRequired bool) ([]string, error) {
	bobfile, errBobfile := readBobfile()
	if errBobfile != nil {
		return nil, errBobfile
	}

	wd, errWd := os.Getwd()
	if errWd != nil {
		return nil, errWd
	}

	builder := findBuilder(bobfile, builderName)
	if builder == nil {
		return nil, ErrBuilderNotFound
	}

	containerName := devContainerName(bobfile, builder.Name)

	var dockerCmd []string
	if isDevContainerRunning(containerName) {
		dockerCmd = append([]string{
			"docker",
			"exec",
			"--interactive",
			"--tty",
			containerName}, builder.DevCommand...)
	} else {
		imageName := builderImageName(bobfile, builder.Name)

		printHeading(fmt.Sprintf("Building builder %s (as %s)", builder.Name, imageName))

		if err := buildBuilder(bobfile, builder); err != nil {
			return nil, err
		}

		dockerCmd = []string{
			"docker",
			"run",
			"--rm",
			"--interactive",
			"--tty",
			"--name", containerName,
			"--volume", wd + "/" + builder.MountSource + ":" + builder.MountDestination,
			"--volume", "/tmp/build:/tmp/build", // cannot map to /tmp because at least apt won't work (permission issues?)
		}

		for _, port := range builder.DevPorts {
			dockerCmd = append(dockerCmd, "--publish", port)
		}

		// inserts ["--env", "FOO"] pairs for each PassEnvs
		var errEnv error
		dockerCmd, errEnv = dockerRelayEnvVars(
			dockerCmd,
			revisionMetadataForDev(),
			false,
			builder.PassEnvs,
			envsAreRequired)
		if errEnv != nil {
			return nil, errEnv
		}

		dockerCmd = append(dockerCmd, imageName)
		dockerCmd = append(dockerCmd, builder.DevCommand...)
	}

	if len(builder.DevPorts) > 0 {
		fmt.Printf("Pro-tip: mapped dev ports: %s\n", strings.Join(builder.DevPorts, ", "))
	}

	for _, proTip := range builder.DevProTips {
		fmt.Printf("Pro-tip: %s\n", proTip)
	}

	return dockerCmd, nil
}

func dev(builderName string, envsAreRequired bool) error {
	dockerCmd, err := devCommand(builderName, envsAreRequired)
	if err != nil {
		return err
	}

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

	cmd := &cobra.Command{
		Use:   "dev [builderName]",
		Short: "Enter builder container in dev mode",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			builderName := "default"
			if len(args) >= 1 {
				builderName = args[0]
			}

			if !dry {
				reactToError(dev(builderName, !norequireEnvs))
			} else {
				dockerCommand, err := devCommand(builderName, !norequireEnvs)
				reactToError(err)

				fmt.Println(strings.Join(dockerCommand, " "))
			}
		},
	}

	cmd.Flags().BoolVarP(&norequireEnvs, "norequire-envs", "n", norequireEnvs, "Don´t error out if not all ENV vars are set")
	cmd.Flags().BoolVarP(&dry, "dry", "", dry, "Just print out the dev command (you may need to do something exotic)")

	return cmd
}
