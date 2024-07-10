package main

import (
	"fmt"
	"strings"
)

func expandTagSpecs(specs []TagSpec, buildCtx *BuildContext, imagePrefix string) []string {
	createTag := func(tag string) string { return fmt.Sprintf("%s:%s", imagePrefix, tag) }

	tags := []string{}

	for _, spec := range specs {
		if !spec.UseIf.Passes(buildCtx) {
			continue
		}

		placeholdersReplaced := strings.ReplaceAll(spec.Pattern, "{rev_short}", buildCtx.RevisionId.RevisionIdShort)
		placeholdersReplaced = strings.ReplaceAll(placeholdersReplaced, "{rev_friendly}", buildCtx.RevisionId.FriendlyRevisionId)

		tags = append(tags, createTag(placeholdersReplaced))
	}

	return tags
}

// backwards compat: model old behaviour on top of newer `TagSpec` facility:
//
// 1. `{rev_friendly}` tag gets always pushed
// 2. `latest` gets pushed if `tag_latest=true` and if we're in default branch
func createBackwardsCompatTagSpecs(maybeTagLatest bool) []TagSpec {
	tags := []TagSpec{
		{
			Pattern: "{rev_friendly}",
		},
	}

	if maybeTagLatest {
		true_ := true

		tags = append(tags, TagSpec{
			Pattern: "latest",
			UseIf: &Condition{
				IsDefaultBranch: &true_,
			},
		})
	}

	return tags
}
