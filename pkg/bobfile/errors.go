package bobfile

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

// tech debt: can't update to newer Go to use this func from gokit
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	} else {
		return b
	}
}
