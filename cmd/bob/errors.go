package main

import (
	"errors"
)

var (
	ErrBuilderNotFound             = errors.New("builder not found")
	ErrBobfileNotFound             = errors.New("turbobob.json does not exist. Run $ bob tools init")
	ErrInitBobfileExists           = errors.New("cannot init; Bobfile already exists")
	ErrUnsupportedBobfileVersion   = errors.New("Unsupported Bobfile version")
	ErrDockerCredsEnvNotSet        = errors.New("DOCKER_CREDS not set")
	ErrInvalidDockerCredsEnvFormat = errors.New("Invalid format for DOCKER_CREDS")
	ErrCiFileAlreadyExists         = errors.New("CI file already exists")
	ErrUnableToParseDockerTag      = errors.New("Unable to parse Docker tag")
	ErrIncorrectFileDescriptionBp  = errors.New("you are not supposed to change FileDescriptionBoilerplate")
	ErrInitingWithBobDevVersion    = errors.New("using dev version of Bob. Bob download URL will be wrong")
)

func envVarMissingErr(envKey string) error {
	return errors.New("ENV var missing: " + envKey)
}
