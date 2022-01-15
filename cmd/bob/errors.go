package main

import (
	"errors"
)

var (
	ErrBuilderNotFound             = errors.New("builder not found")
	ErrBobfileNotFound             = errors.New("turbobob.json does not exist. Run $ bob tools init")
	ErrInitBobfileExists           = errors.New("cannot init; Bobfile already exists")
	ErrUnsupportedBobfileVersion   = errors.New("Unsupported Bobfile version")
	ErrInvalidDockerCredsEnvFormat = errors.New("Invalid format for DOCKER_CREDS")
	ErrUnableToParseDockerTag      = errors.New("Unable to parse Docker tag")
	ErrIncorrectFileDescriptionBp  = errors.New("you are not supposed to change FileDescriptionBoilerplate")
)

func envVarMissingErr(envKey string) error {
	return errors.New("ENV var missing: " + envKey)
}
