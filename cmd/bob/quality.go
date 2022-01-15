package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

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

func qualityCheckFiles(rules []FileQualityRule) error {
	for _, rule := range rules {
		if err := qualityCheckFile(rule); err != nil {
			return fmt.Errorf("quality: file %s: %w", rule.Path, err)
		}
	}

	return nil
}

func qualityCheckFile(rule FileQualityRule) error {
	fileContent, err := os.ReadFile(rule.Path)
	switch {
	case err == nil:
		if rule.MustExist != nil && !*rule.MustExist {
			return errors.New("must not exist (but does)")
		}
	case os.IsNotExist(err):
		if rule.MustExist != nil && *rule.MustExist {
			return errors.New("must exist (but does not)")
		}

		return nil // does not exist and wasn't required to exist => OK
	default: // unexpected error
		return err
	}

	for _, mustContain := range rule.MustContain {
		if !strings.Contains(string(fileContent), mustContain) {
			return fmt.Errorf("must contain '%s' (but does not)", mustContain)
		}
	}

	for _, mustNotContain := range rule.MustNotContain {
		if strings.Contains(string(fileContent), mustNotContain) {
			return fmt.Errorf("must not contain '%s' (but does)", mustNotContain)
		}
	}

	return nil
}
