package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TODO: https://github.com/function61/buildbot/blob/master/bin/buildbot.sh

var zeroTime = time.Time{}

type BuildMetadata struct {
	VcKind             string // git | hg | managedByCi
	RevisionId         string
	RevisionIdShort    string
	FriendlyRevisionId string
}

func revisionMetadataFromFull(revisionId string, timestamp time.Time, vcKind string) *BuildMetadata {
	// https://stackoverflow.com/questions/18134627/how-much-of-a-git-sha-is-generally-considered-necessary-to-uniquely-identify-a
	revisionIdShort := revisionId[0:8]
	friendlyRevId := timestamp.Format("20060102_1504") + "_" + revisionIdShort

	return &BuildMetadata{
		VcKind:             vcKind,
		RevisionId:         revisionId,
		RevisionIdShort:    revisionIdShort,
		FriendlyRevisionId: friendlyRevId,
	}
}

func revisionMetadataForDev() *BuildMetadata {
	return &BuildMetadata{
		VcKind:             "managedByCi", // FIXME
		RevisionId:         "dev",
		RevisionIdShort:    "dev",
		FriendlyRevisionId: "dev",
	}
}

func resolveMetadataFromVersionControl(vc Versioncontrol) (*BuildMetadata, error) {
	revisionId, timestamp, err := vc.Identify()
	if err != nil {
		return nil, err
	}

	return revisionMetadataFromFull(revisionId, timestamp, vc.VcKind()), nil
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
	Identify() (string, time.Time, error)
	VcKind() string
	CloneTo(destination string) error
	Pull() error
	Update(revision string) error
}

type Git struct {
	dir string
}

func (g *Git) VcKind() string {
	return "git"
}

func (g *Git) Identify() (string, time.Time, error) {
	output, err := execWithDir(g.dir, "git", "show", "--no-patch", "--format=%H,%ci", "HEAD")
	if err != nil {
		return output, zeroTime, err
	}

	parts := strings.Split(strings.TrimRight(output, "\r\n"), ",")

	timestamp, errTime := time.Parse("2006-01-02 15:04:05 -0700", parts[1])
	if errTime != nil {
		return "", zeroTime, errTime
	}

	return parts[0], timestamp.UTC(), nil
}

func (g *Git) CloneTo(destination string) error {
	_, err := execWithDir(g.dir, "git", "clone", "--no-checkout", g.dir, destination)
	return err
}

func (g *Git) Pull() error {
	_, err := execWithDir(g.dir, "git", "fetch")
	return err
}

func (g *Git) Update(revision string) error {
	_, err := execWithDir(g.dir, "git", "checkout", "--force", revision)
	return err
}

type Mercurial struct {
	dir string
}

func (g *Mercurial) VcKind() string {
	return "hg"
}

func (m *Mercurial) Identify() (string, time.Time, error) {
	output, err := execWithDir(m.dir, "hg", "log", "--rev", ".", "--template", "{node},{date|isodate}")
	if err != nil {
		return "", zeroTime, err
	}

	parts := strings.Split(output, ",")

	timestamp, errTime := time.Parse("2006-01-02 15:04 -0700", parts[1])
	if errTime != nil {
		return "", zeroTime, errTime
	}

	return parts[0], timestamp.UTC(), nil
}

func (m *Mercurial) CloneTo(destination string) error {
	_, err := execWithDir(m.dir, "hg", "clone", "--noupdate", m.dir, destination)
	return err
}

func (m *Mercurial) Pull() error {
	_, err := execWithDir(m.dir, "hg", "pull")
	return err
}

func (m *Mercurial) Update(revision string) error {
	_, err := execWithDir(m.dir, "hg", "update", "--rev", revision)
	return err
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
