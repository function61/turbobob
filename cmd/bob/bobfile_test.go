package main

import (
	"testing"

	"github.com/function61/gokit/testing/assert"
)

func TestAssertUniqueBuilderNames(t *testing.T) {
	bobfileEmpty := &Bobfile{}
	bobfileUniques := &Bobfile{
		Builders: []BuilderSpec{
			{
				Name: "default",
			},
			{
				Name: "foobar",
			},
		},
	}
	bobfileNonUniques := &Bobfile{
		Builders: []BuilderSpec{
			{
				Name: "foobar",
			},
			{
				Name: "foobar",
			},
		},
	}

	assert.Assert(t, validateBuilders(bobfileEmpty) == nil)
	assert.Assert(t, validateBuilders(bobfileUniques) == nil)
	assert.EqualString(t,
		validateBuilders(bobfileNonUniques).Error(),
		"duplicate builder name: foobar")
}
