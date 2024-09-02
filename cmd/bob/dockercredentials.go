package main

import (
	"errors"
	"os"
	"regexp"

	"github.com/function61/turbobob/pkg/bobfile"
)

type DockerCredentials struct {
	Username string
	Password string
}

type DockerCredentialObtainer interface {
	// can return nil credentials with nil error
	Obtain() (*DockerCredentials, error)
}

type credsFromENV struct{}

var credsFromENVRe = regexp.MustCompile("^([^:]+):(.+)$")

func (d *credsFromENV) Obtain() (*DockerCredentials, error) {
	serialized := os.Getenv("DOCKER_CREDS")
	if serialized == "" {
		return nil, nil
	}

	credsParts := credsFromENVRe.FindStringSubmatch(serialized)
	if len(credsParts) != 3 {
		return nil, bobfile.ErrInvalidDockerCredsEnvFormat
	}

	return &DockerCredentials{
		Username: credsParts[1],
		Password: credsParts[2],
	}, nil
}

func getDockerCredentialsObtainer(dockerImage bobfile.DockerImageSpec) DockerCredentialObtainer {
	if dockerImage.AuthType == nil {
		return &credsFromENV{}
	}

	switch *dockerImage.AuthType {
	case "creds_from_env":
		return &credsFromENV{}
	default:
		panic(errors.New("invalid AuthType: " + *dockerImage.AuthType))
	}
}
