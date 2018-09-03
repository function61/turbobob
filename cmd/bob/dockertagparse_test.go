package main

import (
	"fmt"
	"testing"
)

func TestParseDockerTag(t *testing.T) {
	EqualString(
		t,
		serialize(ParseDockerTag("redis")),
		"registry<> namespace<> repository<redis> tag<>")

	EqualString(
		t,
		serialize(ParseDockerTag("redis:1.2.3.4")),
		"registry<> namespace<> repository<redis> tag<1.2.3.4>")

	EqualString(
		t,
		serialize(ParseDockerTag("joonas/redis:1.2.3.4")),
		"registry<> namespace<joonas> repository<redis> tag<1.2.3.4>")

	EqualString(
		t,
		serialize(ParseDockerTag("docker.io/joonas/redis:1.2.3.4")),
		"registry<docker.io> namespace<joonas> repository<redis> tag<1.2.3.4>")
}

func serialize(tag *DockerTag) string {
	return fmt.Sprintf(
		"registry<%s> namespace<%s> repository<%s> tag<%s>",
		tag.Registry,
		tag.Namespace,
		tag.Repository,
		tag.Tag)
}

func EqualString(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Fatalf("exp=%v; got=%v", expected, actual)
	}
}
