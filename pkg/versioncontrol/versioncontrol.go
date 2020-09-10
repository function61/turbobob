package versioncontrol

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/function61/gokit/os/osutil"
)

func CurrentRevisionId(vc Interface, onlyCommitted bool) (*RevisionId, error) {
	revisionId, revisionTimestamp, err := vc.Identify()
	if err != nil {
		return nil, err
	}

	// https://stackoverflow.com/questions/18134627/how-much-of-a-git-sha-is-generally-considered-necessary-to-uniquely-identify-a
	revisionIdShort := revisionId[0:8]
	friendlyRevId := revisionTimestamp.Format("20060102_1504") + "_" + revisionIdShort

	if !onlyCommitted {
		revisionId += "-uncommitted"
		revisionIdShort += "-uncommitted"
		friendlyRevId = time.Now().Format("20060102_1504") + "_" + revisionIdShort
	}

	return &RevisionId{
		VcKind:             vc.VcKind(),
		RevisionId:         revisionId,
		RevisionIdShort:    revisionIdShort,
		FriendlyRevisionId: friendlyRevId,
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
