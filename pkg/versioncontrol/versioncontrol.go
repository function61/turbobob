package versioncontrol

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/function61/gokit/os/osutil"
)

func CurrentRevisionID(vc Interface, onlyCommitted bool) (*RevisionID, error) {
	revisionID, revisionTimestamp, err := vc.Identify()
	if err != nil {
		return nil, err
	}

	// https://stackoverflow.com/questions/18134627/how-much-of-a-git-sha-is-generally-considered-necessary-to-uniquely-identify-a
	revisionIDShort := revisionID[0:8]
	friendlyRevID := revisionTimestamp.Format("20060102_1504") + "_" + revisionIDShort

	if !onlyCommitted {
		revisionID += "-uncommitted"
		revisionIDShort += "-uncommitted"
		friendlyRevID = time.Now().Format("20060102_1504") + "_" + revisionIDShort
	}

	return &RevisionID{
		VcKind:             vc.VcKind(),
		RevisionID:         revisionID,
		RevisionIDShort:    revisionIDShort,
		FriendlyRevisionID: friendlyRevID,
	}, nil
}

func ForDir(dir string, kind Kind) (Interface, error) {
	switch kind {
	case KindGit:
		return NewGit(dir), nil
	case KindMercurial:
		return NewMercurial(dir), nil
	default:
		return nil, fmt.Errorf("unsupported Kind: %s", kind)
	}
}

func DetectForDirectory(dir string) (Interface, error) {
	isHg, err := osutil.Exists(filepath.Join(dir, ".hg"))
	if err != nil {
		return nil, err
	}

	if isHg {
		return NewMercurial(dir), nil
	}

	isGit, err := osutil.Exists(filepath.Join(dir, ".git"))
	if err != nil {
		return nil, err
	}

	if isGit {
		return NewGit(dir), nil
	}

	return nil, ErrVcMechanismNotIdentified
}
