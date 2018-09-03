package main

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"net/http"
	"os"
	"regexp"
)

type DockerCredentialsObtainer struct {
}

func (d *DockerCredentialsObtainer) IsObtainable() error {
	_, _, err := fromSerializedDockerCredsEnv()
	return err
}

func (d *DockerCredentialsObtainer) Obtain() (*DockerCredentials, error) {
	username, password, err := fromSerializedDockerCredsEnv()
	if err != nil {
		return nil, err
	}

	return &DockerCredentials{
		Username: username,
		Password: password,
	}, nil
}

type DockerCredentials struct {
	Username string
	Password string
}

type AwsEcrCredentialsObtainer struct {
}

func (d *AwsEcrCredentialsObtainer) IsObtainable() error {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		return errors.New("AWS_ACCESS_KEY_ID not set")
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return errors.New("AWS_SECRET_ACCESS_KEY not set")
	}

	return nil
}

func (d *AwsEcrCredentialsObtainer) Obtain() (*DockerCredentials, error) {
	cfg := aws.NewConfig()
	cfg.Credentials = aws.NewStaticCredentialsProvider(
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"")

	// TODO: support other regions as well
	cfg.Region = endpoints.UsEast1RegionID

	ecrClient := ecr.New(*cfg)

	resp, err := ecrClient.GetAuthorizationTokenRequest(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []string{},
	}).Send()
	if err != nil {
		return nil, err
	}

	if len(resp.AuthorizationData) != 1 {
		return nil, errors.New("problem with GetAuthorizationToken response")
	}

	// AuthorizationData is HTTP Basic auth format
	username, password, ok := parseBasicAuthRaw(*resp.AuthorizationData[0].AuthorizationToken)
	if !ok {
		return nil, errors.New("invalid format of AuthorizationToken")
	}

	return &DockerCredentials{
		Username: username,
		Password: password,
	}, nil
}

type DockerCredentialObtainer interface {
	IsObtainable() error
	Obtain() (*DockerCredentials, error)
}

var dockerCredsRe = regexp.MustCompile("^([^:]+):(.+)")

func fromSerializedDockerCredsEnv() (string, string, error) {
	serialized := os.Getenv("DOCKER_CREDS")
	if serialized == "" {
		return "", "", ErrDockerCredsEnvNotSet
	}

	credsParts := dockerCredsRe.FindStringSubmatch(serialized)
	if len(credsParts) != 3 {
		return "", "", ErrInvalidDockerCredsEnvFormat
	}

	return credsParts[1], credsParts[2], nil
}

func getDockerCredentialsObtainer(dockerImage DockerImageSpec) DockerCredentialObtainer {
	switch dockerImage.AuthType {
	case "creds_from_env":
		return &DockerCredentialsObtainer{}
	case "aws_ecr":
		return &AwsEcrCredentialsObtainer{}
	default:
		panic(errors.New("invalid AuthType: " + dockerImage.AuthType))
	}
}

func parseBasicAuthRaw(raw string) (string, string, bool) {
	dummyReq, _ := http.NewRequest("GET", "http://dummy.com/", nil)
	dummyReq.Header.Set("Authorization", "Basic "+raw)
	return dummyReq.BasicAuth()
}
