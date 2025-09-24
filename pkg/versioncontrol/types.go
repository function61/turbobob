package versioncontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrVcMechanismNotIdentified = errors.New("VC mechanism not identified")
)

type Interface interface {
	Identify() (string, time.Time, error)
	VcKind() string
	WithAnotherDir(dir string) Interface
	CloneFrom(source string) error
	Pull() error
	Update(revision string) error
}

type RevisionID struct {
	VcKind             string // git | hg | managedByCi
	RevisionID         string
	RevisionIDShort    string
	FriendlyRevisionID string
}

type Kind string

const (
	KindGit       Kind = "git"
	KindMercurial Kind = "hg"
)

func (s Kind) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(s) + `"`), nil
}

func (s *Kind) UnmarshalJSON(b []byte) error {
	var raw string
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	kind, err := kindFromString(raw)
	if err != nil {
		return err
	}
	*s = kind
	return nil
}

func kindFromString(input string) (Kind, error) {
	switch Kind(input) {
	case KindGit:
		return KindGit, nil
	case KindMercurial:
		return KindMercurial, nil
	default:
		return "", fmt.Errorf("illegal Kind: %s", input)
	}
}
