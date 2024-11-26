package main

import (
	"bytes"
	"testing"

	"github.com/function61/gokit/testing/assert"
)

func TestGithubStepSummaryWriteImages(t *testing.T) {
	output := &bytes.Buffer{}

	assert.Ok(t, githubStepSummaryWriteImagesWithWriter(output, []imageBuildOutput{
		{tag: "fn61/varasto:v1.2.3"},
		{tag: "fn61/varasto-somethingelse:v1.2.3"},
	}))

	//nolint:staticcheck
	assert.EqualString(t, output.String(), "## Image: varasto\n\n```\nfn61/varasto:v1.2.3\n```\n\n\n## Image: varasto-somethingelse\n\n```\nfn61/varasto-somethingelse:v1.2.3\n```\n\n")
}
