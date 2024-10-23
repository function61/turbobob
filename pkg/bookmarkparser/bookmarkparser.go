// Parses code bookmarks from project source code's annotations
package bookmarkparser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/turbobob/pkg/annotationparser"
)

type ID string

var (
	// starting with URL safe characters (to easily represent in URLs) restrictive set is always safer to widen than the other way around.
	// subset of https://stackoverflow.com/a/695469
	idRe = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
)

func (i ID) Validate() error {
	if !idRe.MatchString(string(i)) {
		return fmt.Errorf("bookmark ID '%s' does not match regex '%s'", i, idRe.String())
	}

	return nil
}

type Bookmark struct {
	ID   ID     // (hopefully) permanent ID, semantic "symbol" of the bookmark. i.e. favor naming by the concept rather than directly e.g. function name.
	File string // path relative to repository root
	Line int    // line being referenced
}

// @:{"bookmark": "ParseBookmarks"}
func ParseBookmarks(ctx context.Context, dir string, output io.Writer) error {
	allAnnotations := []annotationparser.Location{}

	// parses annotations, but not yet mapped to bookmarks
	if err := annotationparser.ScanDirectoryFilesRecursively(ctx, dir, func(annotation annotationparser.Location) error {
		allAnnotations = append(allAnnotations, annotation)
		return nil
	}, annotationparser.DefaultParsersForFileTypes); err != nil {
		return err
	}

	bookmarks, err := annotationsToBookmarks(allAnnotations)
	if err != nil {
		return err
	}

	return jsonfile.Marshal(output, bookmarks)
}

func annotationsToBookmarks(annotations []annotationparser.Location) ([]Bookmark, error) {
	withErr := func(err error) ([]Bookmark, error) { return nil, fmt.Errorf("annotationsToBookmarks: %w", err) }

	bookmarks := []Bookmark{}
	for _, annotation := range annotations {
		bookmarkID, hasBookmark := annotation.Annotations["bookmark"]
		if !hasBookmark {
			continue
		}

		id_, ok := bookmarkID.(string)
		if !ok {
			return withErr(errors.New("bookmark ID not string"))
		}

		id := ID(id_)

		if err := id.Validate(); err != nil {
			return withErr(err)
		}

		bookmarks = append(bookmarks, Bookmark{
			ID:   id,
			File: annotation.File,
			Line: annotation.LineTarget(),
		})
	}

	return bookmarks, nil
}
