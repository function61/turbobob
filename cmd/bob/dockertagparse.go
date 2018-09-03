package main

import (
	"regexp"
	"strings"
)

// could also use github.com/docker/distribution/reference but it seems confusing

// thanks https://github.com/mafintosh/docker-parse-image/blob/master/index.js
var parseRe = regexp.MustCompile("^(?:([^\\/]+)\\/)?(?:([^\\/]+)\\/)?([^@:\\/]+)(?:[@:](.+))?$")

type DockerTag struct {
	Registry   string
	Namespace  string
	Repository string
	Tag        string
}

func ParseDockerTag(serialized string) *DockerTag {
	match := parseRe.FindStringSubmatch(serialized)
	if len(match) != 5 {
		return nil
	}

	if match[2] == "" && match[1] != "" && !strings.Contains(match[1], ".") {
		return &DockerTag{
			Registry:   "",
			Namespace:  match[1],
			Repository: match[3],
			Tag:        match[4],
		}
	}

	return &DockerTag{
		Registry:   match[1],
		Namespace:  match[2],
		Repository: match[3],
		Tag:        match[4],
	}
}
