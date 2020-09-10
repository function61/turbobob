package dockertag

import (
	"fmt"
	"testing"

	"github.com/function61/gokit/testing/assert"
)

func TestParseDockerTag(t *testing.T) {
	assert.EqualString(
		t,
		serialize(Parse("redis")),
		"registry<> namespace<> repository<redis> tag<>")

	assert.EqualString(
		t,
		serialize(Parse("redis:1.2.3.4")),
		"registry<> namespace<> repository<redis> tag<1.2.3.4>")

	assert.EqualString(
		t,
		serialize(Parse("joonas/redis:1.2.3.4")),
		"registry<> namespace<joonas> repository<redis> tag<1.2.3.4>")

	assert.EqualString(
		t,
		serialize(Parse("docker.io/joonas/redis:1.2.3.4")),
		"registry<docker.io> namespace<joonas> repository<redis> tag<1.2.3.4>")

	assert.EqualString(
		t,
		serialize(Parse("123456.dkr.ecr.us-east-1.amazonaws.com/joonas.fi-blog")),
		"registry<123456.dkr.ecr.us-east-1.amazonaws.com> namespace<> repository<joonas.fi-blog> tag<>")

	assert.EqualString(
		t,
		serialize(Parse("registry.gitlab.com/function61/project/subcomponent:1.2.3.4")),
		"registry<registry.gitlab.com> namespace<function61> repository<project/subcomponent> tag<1.2.3.4>")

	assert.EqualString(
		t,
		serialize(Parse("")),
		"(failed to parse)")
}

func serialize(tag *Tag) string {
	if tag == nil {
		return "(failed to parse)"
	}

	return fmt.Sprintf(
		"registry<%s> namespace<%s> repository<%s> tag<%s>",
		tag.Registry,
		tag.Namespace,
		tag.Repository,
		tag.Tag)
}
