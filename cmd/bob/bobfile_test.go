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

	assert.Assert(t, assertUniqueBuilderNames(bobfileEmpty) == nil)
	assert.Assert(t, assertUniqueBuilderNames(bobfileUniques) == nil)
	assert.EqualString(t,
		assertUniqueBuilderNames(bobfileNonUniques).Error(),
		"duplicate builder name: foobar")
}
