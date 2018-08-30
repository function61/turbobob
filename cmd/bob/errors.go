package main

import (
	"errors"
)

var (
	ErrBuilderNotFound           = errors.New("builder not found")
	ErrCiRevisionIdEnvNotSet     = errors.New("CI_REVISION_ID not set")
	ErrVcMechanismNotIdentified  = errors.New("VC mechanism not identified")
	ErrBobfileNotFound           = errors.New("bob.json does not exist. Run $ bob init")
	ErrInitBobfileExists         = errors.New("cannot init; Bobfile already exists")
	ErrUnsupportedBobfileVersion = errors.New("Unsupported Bobfile version")
)

func unknownCommand(command string) error {
	return errors.New("unknown command: " + command)
}

func envVarMissingErr(envKey string) error {
	return errors.New("ENV var missing: " + envKey)
}
