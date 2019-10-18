package dockertag

import (
	"regexp"
	"strings"
)

const DockerHubHostname = "docker.io"

// could also use github.com/docker/distribution/reference but it seems confusing

// thanks https://github.com/mafintosh/docker-parse-image/blob/master/index.js
// + https://github.com/mafintosh/docker-parse-image/pull/2
var parseRe = regexp.MustCompile("^(?:([^\\/]+)\\/)?(?:([^\\/]+)\\/)?([^@:]+)(?:[@:](.+))?$")

type Tag struct {
	Registry   string
	Namespace  string
	Repository string
	Tag        string
}

func Parse(serialized string) *Tag {
	match := parseRe.FindStringSubmatch(serialized)
	if len(match) != 5 {
		return nil
	}

	if match[2] == "" && match[1] != "" && !strings.Contains(match[1], ".") {
		return &Tag{
			Registry:   "",
			Namespace:  match[1],
			Repository: match[3],
			Tag:        match[4],
		}
	}

	return &Tag{
		Registry:   match[1],
		Namespace:  match[2],
		Repository: match[3],
		Tag:        match[4],
	}
}
