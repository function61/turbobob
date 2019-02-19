package main

import (
	"os"
	"os/exec"
)

func passthroughStdoutAndStderr(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

func isEnvVarPresent(key string) bool {
	return os.Getenv(key) != ""
}
