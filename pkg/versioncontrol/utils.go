package versioncontrol

import (
	"fmt"
	"os/exec"
	"time"
)

var zeroTime = time.Time{}

func execWithDir(dir string, args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, output)
	}

	return string(output), nil
}
