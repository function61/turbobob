package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/function61/gokit/os/osutil"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func openProjectHomepageEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:     "www",
		Aliases: []string{"gh"},
		Short:   "Open project homepage / GitHub repo in browser",
		Hidden:  true,
		Args:    cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			osutil.ExitIfError(func() error {
				repo, err := gitHubRepoRefFromGitNonNil()
				if err != nil {
					return err
				}

				// not interested in browser output
				browser.Stdout = io.Discard
				browser.Stderr = io.Discard

				return browser.OpenURL(githubURL(*repo))
			}())
		},
	}
}

func gitHubRepoRefFromGitNonNil() (*githubRepoRef, error) {
	ref, err := gitHubRepoRefFromGit()
	if err != nil {
		return nil, err
	}

	if ref == nil {
		return nil, errors.New("unable to resolve GitHub organization/repo name")
	}

	return ref, err
}

// NOTE: returns nil, nil if not a Git repo OR no origin specified OR not a GitHub origin
func gitHubRepoRefFromGit() (*githubRepoRef, error) {
	conf, err := os.ReadFile(".git/config")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else { // some other error
			return nil, err
		}
	}

	// dirty
	originParseRe := regexp.MustCompile(`url = git@github.com:(.+)/(.+).git`)

	matches := originParseRe.FindStringSubmatch(string(conf))
	if matches == nil {
		return nil, nil
	}

	org, repo := matches[1], matches[2]

	return &githubRepoRef{org, repo}, nil
}

type githubRepoRef struct {
	org  string
	repo string
}

func githubURL(repo githubRepoRef) string {
	return fmt.Sprintf("https://github.com/%s/%s", repo.org, repo.repo)
}
