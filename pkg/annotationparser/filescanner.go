package annotationparser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
)

func ScanDirectoryFilesRecursively(ctx context.Context, dir string, processAnnotation func(Location) error, parsers map[string]Parser) error {
	if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if isIgnoredEntryName(info.Name()) {
			return fs.SkipDir
		}

		if info.IsDir() { // only process files
			return nil
		}

		ext := filepath.Ext(path)
		if parser, found := parsers[ext]; found {
			fullPath := filepath.Join(dir, path)

			if err := parser.Parse(ctx, fullPath, processAnnotation); err != nil {
				return fmt.Errorf("Parse %s: %w", fullPath, err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func isIgnoredEntryName(name string) bool {
	return name == ".git"
}

func scanLineBasedContentForRegexMatches(
	content io.Reader,
	re *regexp.Regexp,
	processResult func(match string, fullLine string, lineNumber int, lineOffset int) error,
) error {
	// scan line by line to be able to keep track of line numbers and annotation match byte offsets more easily.
	//
	// NOTE: docs advise to use a higher-level scanner but returned lines don't contain the line terminators
	// and since the terminator can be "\r\n" or "\n" we can't calculate line byte offsets unless we know the exact line.
	lineScanner := bufio.NewReader(content)

	lineNumber := 1
	lineOffset := 0

	for {
		// line can end in "\n" (Unix convention) "\r\n" (Windows convention).
		// both conveniently end in "\n" so that's what we look for.
		line, err := lineScanner.ReadBytes('\n')
		isEOF := err == io.EOF
		if err != nil && !isEOF { // last line has eof set but we need to process its possible content first before stopping
			return err
		}

		annotationsMatch := re.FindStringSubmatch(string(line))

		if annotationsMatch != nil {
			if err := processResult(annotationsMatch[1], annotationsMatch[0], lineNumber, lineOffset); err != nil {
				return err
			}
		}

		lineNumber++
		lineOffset += len(line)

		if isEOF {
			return nil
		}
	}
}
