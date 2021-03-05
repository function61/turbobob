package main

import (
	"fmt"

	"github.com/function61/gokit/os/osutil"
)

func qualityCheckFilesThatShouldExist(filesThatShouldExist []string) error {
	for _, fileThatShouldExist := range filesThatShouldExist {
		exists, err := osutil.Exists(fileThatShouldExist)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("quality: file that should, does not exist: %s", fileThatShouldExist)
		}
	}

	return nil
}
