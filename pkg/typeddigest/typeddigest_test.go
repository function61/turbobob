package typeddigest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/function61/gokit/testing/assert"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{
			"sha256:d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			"ok",
		},
		{
			"sha256:88",
			"ERROR: typeddigest.Parse: wrong digest size: expected 32; got 1",
		},
		{
			"md5:d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			"ERROR: typeddigest.Parse: unsupported algorithm: md5",
		},
		{
			"",
			"ERROR: typeddigest.Parse: bad format, '<alg>:' prefix not found",
		},
		{
			"sha256:nothex",
			"ERROR: typeddigest.Parse: encoding/hex: invalid byte: U+006E 'n'",
		},
	} {
		tc := tc // pin
		t.Run(tc.input, func(t *testing.T) {
			th, err := Parse(tc.input)
			asOutput := func() string {
				if err != nil {
					return fmt.Sprintf("ERROR: %v", err)
				} else {
					// test stability
					//nolint:staticcheck // cannot upgrade to generics yet
					assert.EqualString(t, th.String(), tc.input)
					return "ok"
				}
			}()

			//nolint:staticcheck // cannot upgrade to generics yet
			assert.EqualString(t, asOutput, tc.output)
		})
	}
}

func TestSha256(t *testing.T) {
	th, err := Sha256(strings.NewReader("The quick brown fox jumps over the lazy dog"))
	assert.Ok(t, err)

	//nolint:staticcheck // cannot upgrade to generics yet
	assert.EqualString(t, th.String(), "sha256:d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
}
