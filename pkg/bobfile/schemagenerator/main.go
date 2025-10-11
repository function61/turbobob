package main

import (
	"errors"
	"os"
	"runtime/debug"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/invopop/jsonschema"
)

//go:generate go run .

func main() {
	if err := bobfileToJSONSchema(); err != nil {
		panic(err)
	}
}

func bobfileToJSONSchema() error {
	reflector := &jsonschema.Reflector{}
	if err := insertGoCodeCommentsToReflector(reflector); err != nil {
		return err
	}

	bobfileJSONSchema := reflector.Reflect(&bobfile.Bobfile{})
	return jsonfile.Write("../bobfile.schema.json", bobfileJSONSchema)
}

func insertGoCodeCommentsToReflector(reflector *jsonschema.Reflector) error {
	// NOTE: need this horrible hack to change dir to root (the `AddGoComments` doesn't seem to work unless we're in root)
	return inDifferentWorkdir("../../../", func() error {
		modulePath, err := resolveModulePath()
		if err != nil {
			return err
		}

		// needs source code to be able to access the comments
		return reflector.AddGoComments(modulePath, "./")
	})
}

// looks like "github.com/myorg/myrepo". resolve dynamically to DRY.
func resolveModulePath() (string, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("failed to read build info")
	}

	return buildInfo.Main.Path, nil
}

func inDifferentWorkdir(workdir string, work func() error) error {
	previous, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(workdir); err != nil {
		return err
	}

	defer func() {
		_ = os.Chdir(previous)
	}()

	return work()
}
