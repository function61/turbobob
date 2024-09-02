// License tooling
package license

import (
	"errors"
	"fmt"
	"os"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/function61/turbobob/pkg/typeddigest"
	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	assign := false

	cmd := &cobra.Command{
		Use:   "license",
		Short: "Autodetect license",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(autodetectTool(assign))
		},
	}

	cmd.Flags().BoolVarP(&assign, "assign", "", assign, "Assign the detected license for this project metadata")

	cmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate that autodetected license is the same as what is stored in project metadata",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(func() error {
				projectFile, err := bobfile.Read()
				if err != nil {
					return err
				}

				if projectFile.Meta.License == nil {
					return errors.New("project does not specify a license")
				}

				return licenseValidate(*projectFile.Meta.License)
			}())
		},
	})

	return cmd
}

func autodetectTool(assign bool) error {
	lic, err := autodetect()
	if err != nil {
		return err
	}

	if assign {
		// can't allow reading from legacy location, as writing the file would write to different location
		projectFile, err := bobfile.ReadWithAllowLegacyOption(false)
		if err != nil {
			return err
		}

		projectFile.Meta.License = lic

		return jsonfile.Write(bobfile.Name, projectFile)
	} else {
		return jsonfile.Marshal(os.Stdout, lic)
	}
}

func autodetect() (*bobfile.LicenseInfo, error) {
	withErr := func(err error) (*bobfile.LicenseInfo, error) { return nil, fmt.Errorf("autodetect: %w", err) }

	licenseFiler, err := filer.FromDirectory(".")
	if err != nil {
		return withErr(err)
	}

	const confidenceThreshold = 0.97

	licenses, err := licensedb.Detect(licenseFiler)
	if err != nil {
		return withErr(err)
	}

	type matchItem struct {
		license    string // SPDX id
		file       string
		confidence float64
	}

	matches := []matchItem{}
	for licenseKey, match := range licenses {
		if match.Confidence < confidenceThreshold {
			continue
		}

		matches = append(matches, matchItem{ // transform wonky data structure to simpler one
			license:    licenseKey,
			file:       match.File,
			confidence: float64(match.Confidence),
		})
	}

	if l := len(matches); l != 1 {
		return withErr(fmt.Errorf("expected one; got %d", l))
	}
	match := matches[0]

	licenseFile, err := os.Open(match.file)
	if err != nil {
		return withErr(err)
	}
	defer licenseFile.Close()

	licenseDigest, err := typeddigest.Sha256(licenseFile)
	if err != nil {
		return withErr(err)
	}

	return &bobfile.LicenseInfo{
		Expression: match.license,
		Rationale: bobfile.LicenseInfoRationale{
			Kind: bobfile.LicenseInfoRationaleKindAutodetectedFromFile,
			AutodetectedFromFile: &bobfile.LicenseInfoRationaleAutodetectedFromFile{
				File:       match.file,
				Digest:     licenseDigest.String(),
				Confidence: float64(match.confidence),
			},
		},
	}, nil
}

func licenseValidate(lic bobfile.LicenseInfo) error {
	withErr := func(err error) error { return fmt.Errorf("licenseValidate: %w", err) }

	switch lic.Rationale.Kind {
	case bobfile.LicenseInfoRationaleKindDefinedManually:
		// manually is manually - there's no validating it automatically
		return nil
	case bobfile.LicenseInfoRationaleKindAutodetectedFromFile:
		rationale := *lic.Rationale.AutodetectedFromFile

		expected, err := typeddigest.Parse(rationale.Digest)
		if err != nil {
			return withErr(err)
		}

		f, err := os.Open(rationale.File)
		if err != nil {
			return withErr(err)
		}
		defer f.Close()

		actual, err := typeddigest.DigesterForAlgOf(expected)(f)
		if err != nil {
			return withErr(err)
		}

		if !actual.Equal(expected) {
			return withErr(errors.New("license source file changed - must re-detect"))
		}

		return nil
	default:
		return withErr(fmt.Errorf("validate: unsupported kind: %s", lic.Rationale.Kind))
	}
}
