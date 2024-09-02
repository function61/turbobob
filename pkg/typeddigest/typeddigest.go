// A digest that contains the algoritm as a prefix. Example `sha256:d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592`
package typeddigest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	algSha256 = "sha256"
)

type Hash struct {
	alg    string // "sha256"
	digest []byte
}

func (h *Hash) String() string {
	return fmt.Sprintf("%s:%x", h.alg, h.digest)
}

func (h *Hash) Equal(other *Hash) bool {
	return h.alg == other.alg && bytes.Equal(h.digest, other.digest)
}

func Parse(val string) (*Hash, error) {
	withErr := func(err error) (*Hash, error) { return nil, fmt.Errorf("typeddigest.Parse: %w", err) }

	pos := strings.Index(val, ":")
	if pos == -1 {
		return withErr(errors.New("bad format, '<alg>:' prefix not found"))
	}

	alg := val[:pos]
	digestHex := val[pos+1:]

	if alg != algSha256 {
		return withErr(fmt.Errorf("unsupported algorithm: %s", alg))
	}

	digest, err := hex.DecodeString(digestHex)
	if err != nil {
		return withErr(err)
	}

	if expectedSize := sha256.Size; len(digest) != expectedSize {
		return withErr(fmt.Errorf("wrong digest size: expected %d; got %d", expectedSize, len(digest)))
	}

	return &Hash{alg, digest}, nil
}

func DigesterForAlgOf(other *Hash) func(io.Reader) (*Hash, error) {
	switch other.alg {
	case algSha256:
		return Sha256
	default:
		return func(io.Reader) (*Hash, error) {
			return nil, fmt.Errorf("typeddigest.DigesterForAlgOf: unsupported algorithm: %s", other.alg)
		}
	}
}

func Sha256(input io.Reader) (*Hash, error) {
	withErr := func(err error) (*Hash, error) { return nil, fmt.Errorf("typeddigest.Sha256: %w", err) }

	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return withErr(err)
	}

	return &Hash{algSha256, hash.Sum(nil)}, nil
}
