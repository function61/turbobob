package main

import (
	"fmt"
	"strings"
)

type builderUsesType int

const (
	builderUsesTypeImage builderUsesType = iota
	builderUsesTypeDockerfile
)

func parseBuilderUsesType(usesSerialized string) (builderUsesType, string, error) {
	if strings.HasPrefix(usesSerialized, "docker://") {
		return builderUsesTypeImage, usesSerialized[len("docker://"):], nil
	} else if strings.HasPrefix(usesSerialized, "dockerfile://") {
		return builderUsesTypeDockerfile, usesSerialized[len("dockerfile://"):], nil
	}

	return 0, "", fmt.Errorf("unsupported Uses format: %s", usesSerialized)
}
