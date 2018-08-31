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

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		// unknown error. maybe error accessing FS?
		return false, err
	}

	return true, nil
}
