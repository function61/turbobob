package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TODO: https://github.com/function61/buildbot/blob/master/bin/buildbot.sh

type BuildMetadata struct {
	VcKind             string // git | hg | managedByCi
	RevisionId         string
	RevisionIdShort    string
	FriendlyRevisionId string
}

func revisionMetadataFromFull(revisionId string, vcKind string) *BuildMetadata {
	revisionIdShort := revisionId[0:16]
	friendlyRevId := time.Now().Format("20060102_1504") + "_" + revisionIdShort

	return &BuildMetadata{
		VcKind:             vcKind,
		RevisionId:         revisionId,
		RevisionIdShort:    revisionIdShort,
		FriendlyRevisionId: friendlyRevId,
	}
}

func resolveMetadataFromVersionControl() (*BuildMetadata, error) {
	wd, errWd := os.Getwd()
	if errWd != nil {
		return nil, errWd
	}

	vc, errVcDetermine := determineVcForDirectory(wd)
	if errVcDetermine != nil {
		return nil, errVcDetermine
	}

	revisionId, err := vc.Identify()
	if err != nil {
		return nil, err
	}

	return revisionMetadataFromFull(revisionId, vc.VcKind()), nil
}

func determineVcForDirectory(dir string) (Versioncontrol, error) {
	isHg, err := fileExists(filepath.Join(dir, ".hg"))
	if err != nil {
		return nil, err
	}

	if isHg {
		return &Mercurial{
			dir: dir,
		}, nil
	}

	isGit, err := fileExists(filepath.Join(dir, ".git"))
	if err != nil {
		return nil, err
	}

	if isGit {
		return &Git{
			dir: dir,
		}, nil
	}

	return nil, ErrVcMechanismNotIdentified
}

type Versioncontrol interface {
	Identify() (string, error)
	VcKind() string
}

type Git struct {
	dir string
}

func (g *Git) VcKind() string {
	return "git"
}

func (g *Git) Identify() (string, error) {
	output, err := execWithDir(g.dir, "git", "rev-parse", "HEAD")
	if err != nil {
		return output, err
	}

	return strings.TrimRight(output, "\r\n"), nil
}

type Mercurial struct {
	dir string
}

func (g *Mercurial) VcKind() string {
	return "hg"

}
func (m *Mercurial) Identify() (string, error) {
	return execWithDir(m.dir, "hg", "log", "--rev", ".", "--template", "{node}")
}

func execWithDir(dir string, args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
