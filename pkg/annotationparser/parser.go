// Parses annotations from source code
package annotationparser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

type Parser interface {
	Parse(ctx context.Context, path string, processAnnotation func(Location) error) error
}

type Annotations map[string]interface{}

// the location of the annotation in a file
type Location struct {
	File        string
	Line        int // the line the annotation is found from. (the line the annotation describes is usually next line)
	Offset      int
	Annotations Annotations
}

func (a Location) LineTarget() int {
	return a.Line + 1
}

var (
	// parses lines that look like `// @:{"key": "value"}`.
	// whitespace is allowed to precede `//`.
	doubleSlashAnnotationParser = &annotationParserFromRegex{regexp.MustCompile(`^[ \t]*// @:([^\n]+)`)}

	DefaultParsersForFileTypes = map[string]Parser{
		/*	Annotation format for Go

			established keys are:

			//go:generaste
			//go:embed
			//go:build
			//nolint:<lintername>
			//easyjson:json

			hence the syntax should be "//" + prefix + ":"

			for prefix seems we cannot use symbols like "@" or "_" because `$ goimports` indents to "// ",
			so must choose alphanumeric keys.
		*/
		".go": doubleSlashAnnotationParser,
	}
)

// uses a regex to parse annotations from a line
type annotationParserFromRegex struct {
	annotationsRe *regexp.Regexp
}

func (g *annotationParserFromRegex) Parse(ctx context.Context, path string, processAnnotation func(Location) error) error {
	// unfortunately we need to buffer this as there is no `FindAllStringSubmatch` that operates on streams
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return scanLineBasedContentForRegexMatches(file, g.annotationsRe, func(match string, fullLine string, lineNumber int, lineOffset int) error {
		annots := Annotations{}

		if err := json.Unmarshal([]byte(match), &annots); err != nil {
			return fmt.Errorf("%w\ninput:\n%s", err, fullLine)
		}

		if err := processAnnotation(Location{
			File:        path,
			Line:        lineNumber,
			Offset:      lineOffset,
			Annotations: annots,
		}); err != nil {
			return err
		}

		return nil
	})
}
