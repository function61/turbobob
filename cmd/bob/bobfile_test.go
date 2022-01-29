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

	assert.Ok(t, validateBuilders(bobfileEmpty))
	assert.Ok(t, validateBuilders(bobfileUniques))
	assert.EqualString(t,
		validateBuilders(bobfileNonUniques).Error(),
		"duplicate builder name: foobar")
}

func TestConsentToBreakage(t *testing.T) {
	properConsent := &Bobfile{
		Builders: []BuilderSpec{
			{
				Name: "foobar",
				Commands: BuilderCommands{
					Prepare: []string{"foo"},
				},
			},
		},
		Experiments: experiments{
			PrepareStep: true,
		},
	}

	missingConsent := &Bobfile{
		Builders: []BuilderSpec{
			{
				Name: "foobar",
				Commands: BuilderCommands{
					Prepare: []string{"foo"},
				},
			},
		},
	}

	assert.Ok(t, validateBuilders(properConsent))
	assert.EqualString(t, validateBuilders(missingConsent).Error(), "foobar: you need to opt-in to prepare_step experiment")
}
