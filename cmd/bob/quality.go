package main

import (
	"fmt"
	"strings"

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

func qualityCheckFilesThatShouldNotExist(filesThatShouldNotExist []string) error {
	for _, fileThatShouldNotExist := range filesThatShouldNotExist {
		exists, err := osutil.Exists(fileThatShouldNotExist)
		if err != nil {
			return err
		}

		if exists {
			return fmt.Errorf("quality: file should not exist: %s", fileThatShouldNotExist)
		}
	}

	return nil
}

func qualityCheckBuilderUsesExpect(rules map[string]string, bobfile *Bobfile) error {
	for _, builder := range bobfile.Builders {
		for substring, expectFull := range rules {
			if strings.Contains(builder.Uses, substring) && builder.Uses != expectFull {
				return fmt.Errorf(
					"quality: outdated builder %s\n                 expected %s",
					builder.Uses,
					expectFull)
			}
		}
	}

	return nil
}
