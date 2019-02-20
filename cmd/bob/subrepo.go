package main

import (
	"fmt"
	"github.com/function61/gokit/fileexists"
)

func ensureSubrepoCloned(destination string, subrepo SubrepoSpec) error {
	exists, err := fileexists.Exists(destination)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	printHeading(fmt.Sprintf("Cloning subrepo %s -> %s", subrepo.Source, destination))

	repo, err := vcForDir(destination, subrepo.Kind)
	if err != nil {
		return err
	}

	if err := repo.CloneFrom(subrepo.Source); err != nil {
		return fmt.Errorf("CloneFrom: %v", err)
	}

	if err := repo.Update(subrepo.Revision); err != nil {
		return fmt.Errorf("Update: %v", err)
	}

	return nil
}
