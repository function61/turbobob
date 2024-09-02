package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/function61/turbobob/pkg/bobfile"
)

func qualityCheckBuilderUsesExpect(rules map[string]string, bobfile *bobfile.Bobfile) error {
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
		if !conditionsPass(rule.Conditions) { // rule not in use because didn't pass all conditions
			continue
		}

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

func conditionsPass(conditions []QualityRuleCondition) bool {
	for _, condition := range conditions {
		conditionPasses := func(matches bool) bool {
			if condition.Enable && !matches { // condition disqualifies if NOT match
				return false
			} else if !condition.Enable && matches { // condition disqualifies if DOES match
				return false
			}

			return true
		}

		if origin := condition.RepoOrigin; origin != "" {
			// "*foo*" => "foo"
			originWildcard := strings.Trim(origin, "*")
			if len(originWildcard) != len(origin)-2 { // stupidest way to check it had * at start AND end
				panic("repo_origin needs to be in form '*foobar*'")
			}

			ref := getGithubRepoRef()

			originMatches := func() bool {
				if ref != nil {
					return strings.Contains(githubURL(*ref), originWildcard)
				} else {
					return false
				}
			}()

			if !conditionPasses(originMatches) {
				return false
			}
		}
	}

	return true
}

// lazily read gitHubRepoRefFromGit() just once
func getGithubRepoRef() *githubRepoRef {
	if githubRepoRefSingleton == nil {
		var err error
		githubRepoRefSingleton, err = gitHubRepoRefFromGit()
		if err != nil {
			panic(err)
		}
	}

	return githubRepoRefSingleton
}

var githubRepoRefSingleton *githubRepoRef
