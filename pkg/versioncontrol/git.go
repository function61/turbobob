package versioncontrol

import (
	"strings"
	"time"
)

type Git struct {
	dir string
}

func NewGit(dir string) Interface {
	return &Git{dir: dir}
}

func (g *Git) WithAnotherDir(dir string) Interface {
	return &Git{dir: dir}
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

func (g *Git) CloneFrom(source string) error {
	// cannot set dir, because target dir does not yet exist
	_, err := execWithDir("", "git", "clone", "--no-checkout", source, g.dir)
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
