package main

import (
	"fmt"
	"testing"

	"github.com/function61/gokit/testing/assert"
	"github.com/function61/turbobob/pkg/versioncontrol"
)

func TestExpandTagSpecs(t *testing.T) {
	true_ := true

	exampleRevision := &versioncontrol.RevisionId{
		RevisionId:         "df9e1a0b32d41977b49742d57702fbae0392c49a",
		RevisionIdShort:    "df9e1a0b",
		FriendlyRevisionId: "20240616_0924_df9e1a0b",
	}

	defaultBuildContext := &BuildContext{RevisionId: exampleRevision}

	for _, tc := range []struct {
		input        TagSpec
		buildContext *BuildContext
		output       string
	}{
		{
			input: TagSpec{
				Pattern: "latest",
			},
			buildContext: &BuildContext{
				RevisionId:      exampleRevision,
				IsDefaultBranch: true,
			},
			output: "[joonas:latest]",
		},
		{
			input: TagSpec{
				Pattern: "latest",
				UseIf: &Condition{
					IsDefaultBranch: &true_,
				},
			},
			buildContext: &BuildContext{
				RevisionId:      exampleRevision,
				IsDefaultBranch: false,
			},
			output: "[]",
		},
		{
			input: TagSpec{
				Pattern: "sha-{rev_short}",
			},
			buildContext: defaultBuildContext,
			output:       "[joonas:sha-df9e1a0b]",
		},
		{
			input: TagSpec{
				Pattern: "{rev_friendly}",
			},
			buildContext: defaultBuildContext,
			output:       "[joonas:20240616_0924_df9e1a0b]",
		},
	} {
		tc := tc // pin

		t.Run(tc.output, func(t *testing.T) {
			tags := expandTagSpecs([]TagSpec{tc.input}, tc.buildContext, "joonas")
			assert.Equal(t, fmt.Sprintf("%v", tags), tc.output)
		})
	}
}
