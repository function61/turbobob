package main

import (
	"fmt"

	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/function61/turbobob/pkg/versioncontrol"
)

func ensureSubrepoCloned(destination string, subrepo bobfile.SubrepoSpec) error {
	exists, err := osutil.Exists(destination)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	printHeading(fmt.Sprintf("Cloning subrepo %s -> %s", subrepo.Source, destination))

	repo, err := versioncontrol.ForDir(destination, subrepo.Kind)
	if err != nil {
		return err
	}

	if err := repo.CloneFrom(subrepo.Source); err != nil {
		return fmt.Errorf("CloneFrom: %w", err)
	}

	if err := repo.Update(subrepo.Revision); err != nil {
		//nolint:staticcheck
		return fmt.Errorf("Update: %w", err)
	}

	return nil
}
