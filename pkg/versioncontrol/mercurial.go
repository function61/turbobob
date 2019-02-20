package versioncontrol

import (
	"strings"
	"time"
)

type Mercurial struct {
	dir string
}

func NewMercurial(dir string) Interface {
	return &Mercurial{dir: dir}
}

func (g *Mercurial) VcKind() string {
	return "hg"
}

func (g *Mercurial) WithAnotherDir(dir string) Interface {
	return &Mercurial{dir: dir}
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

func (m *Mercurial) CloneFrom(source string) error {
	// cannot set dir, because target dir does not yet exist
	_, err := execWithDir("", "hg", "clone", "--noupdate", source, m.dir)
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
