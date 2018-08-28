package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func dev(bobfile *Bobfile, args []string) error {
	builderName := "default"

	if len(args) == 1 {
		builderName = args[0]
	} else if len(args) > 1 {
		return errors.New("Usage: <builder>")
	}

	wd, errWd := os.Getwd()
	if errWd != nil {
		return errWd
	}

	builder := findBuilder(bobfile, builderName)
	if builder == nil {
		return ErrBuilderNotFound
	}

	imageName := builderImageName(bobfile, builder.Name)

	printHeading(fmt.Sprintf("Building builder %s (as %s)", builder.Name, imageName))

	if err := buildBuilder(bobfile, builder); err != nil {
		return err
	}

	containerName := devContainerName(bobfile, builder.Name)

	devCommand := builder.DevCommandOrDefaultToBash()

	var dockerCmd []string
	if isDevContainerRunning(containerName) {
		dockerCmd = append([]string{
			"docker",
			"exec",
			"--interactive",
			"--tty",
			containerName}, devCommand...)
	} else {
		imageName := builderImageName(bobfile, builder.Name)

		metadata, errMetadata := resolveMetadataFromVersionControl()
		if errMetadata != nil {
			return errMetadata
		}

		dockerCmd = []string{
			"docker",
			"run",
			"--rm",
			"--interactive",
			"--tty",
			"--name", containerName,
			"--volume", wd + ":" + builder.MountDestinationOrDefaultToApp(),
			"--volume", "/tmp/bob-tmp:/tmp",
		}

		// inserts ["--env", "FOO"] pairs for each PassEnvs
		var errEnv error
		dockerCmd, errEnv = dockerRelayEnvVars(dockerCmd, metadata, builder.PassEnvs)
		if errEnv != nil {
			return errEnv
		}

		dockerCmd = append(dockerCmd, imageName)
		dockerCmd = append(dockerCmd, devCommand...)
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
