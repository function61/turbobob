package main

// Guess turbobob.json content based on Dockerfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/function61/gokit/builtin"
	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func initGuessFromDockerfile() error {
	workdir, err := os.Getwd()
	if err != nil {
		return err
	}

	projectName := filepath.Base(workdir) // guess

	stages, err := parseDockerfile("Dockerfile")
	if err != nil {
		return ErrorWrap("parseDockerfile", err)
	}

	builderStageIdx, has := instructions.HasStage(stages, "builder")
	if !has {
		return errors.New("expecting stage named 'builder' to be present; was not")
	}

	builderStage := stages[builderStageIdx]

	copyCommand, runCommand, err := extractOneCopyAndRunCommand(builderStage)
	if err != nil {
		return ErrorWrap("extractOneCopyAndRunCommand", err)
	}

	if len(copyCommand.SourcePaths) != 1 || copyCommand.SourcePaths[0] != "." {
		return fmt.Errorf("COPY source must be '.'; got %v", copyCommand.SourcePaths)
	}

	if copyCommand.From != "" {
		return fmt.Errorf("COPY must not have FROM; had '%s'", copyCommand.From)
	}

	buildCommand, err := func() ([]string, error) {
		if runCommand.PrependShell {
			if len(runCommand.CmdLine) != 1 {
				return nil, fmt.Errorf("with RUN shell only one arg is expected; got %d", len(runCommand.CmdLine))
			}

			script := runCommand.CmdLine[0]

			return []string{"bash", "-c", script}, nil
		} else { // exec form
			return runCommand.CmdLine, nil
		}
	}()
	if err != nil {
		return err
	}

	return writeBobfileIfNotExists(bobfile.Bobfile{
		FileDescriptionBoilerplate: bobfile.FileDescriptionBoilerplate,
		VersionMajor:               bobfile.CurrentVersionMajor,
		ProjectName:                projectName,
		Builders: []bobfile.BuilderSpec{
			{
				Name:             "default",
				Uses:             "docker://" + builderStage.BaseName,
				MountDestination: copyCommand.DestPath,
				Commands: bobfile.BuilderCommands{
					Build: buildCommand,
					Dev:   []string{"bash"}, // just a guess
				},
			},
		},
	})
}

func parseDockerfile(path string) ([]instructions.Stage, error) {
	dockerfileFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dockerfileFile.Close()

	dockerfileAST, err := parser.Parse(dockerfileFile)
	if err != nil {
		return nil, err
	}

	stages, _, err := instructions.Parse(dockerfileAST.AST, nil)
	if err != nil {
		return nil, err
	}

	return stages, err
}

func extractOneCopyAndRunCommand(stage instructions.Stage) (*instructions.CopyCommand, *instructions.RunCommand, error) {
	copies := []*instructions.CopyCommand{}
	runs := []*instructions.RunCommand{}

	for _, command := range stage.Commands {
		switch el := command.(type) {
		case *instructions.CopyCommand:
			copies = append(copies, el)
		case *instructions.RunCommand:
			runs = append(runs, el)
		default:
			return nil, nil, fmt.Errorf("unknown command: %s", command.Name())
		}
	}

	if len(copies) != 1 {
		return nil, nil, fmt.Errorf("expected exactly one COPY instruction; got %d", len(copies))
	}

	if len(runs) != 1 {
		return nil, nil, fmt.Errorf("expected exactly one RUN instruction; got %d", len(runs))
	}

	return copies[0], runs[0], nil
}
