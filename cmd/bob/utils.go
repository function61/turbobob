package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/function61/turbobob/pkg/bobfile"
	"github.com/function61/turbobob/pkg/dockertag"
)

func passthroughStdoutAndStderr(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

func isEnvVarPresent(key string) bool {
	return os.Getenv(key) != ""
}

func envVarMissingErr(envKey string) error {
	return errors.New("ENV var missing: " + envKey)
}

func allDevShellCommands(devShellCommands []bobfile.DevShellCommand) []string {
	commands := []string{}
	for _, command := range devShellCommands {
		commands = append(commands, command.Command)
	}

	return commands
}

func runningInGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func must(input builderUsesType, _ string, err error) builderUsesType {
	if err != nil {
		panic(err)
	}
	return input
}

func findBuilder(projectFile *bobfile.Bobfile, builderName string) (*bobfile.BuilderSpec, error) {
	for _, builder := range projectFile.Builders {
		if builder.Name == builderName {
			return &builder, nil
		}
	}

	return nil, bobfile.ErrBuilderNotFound
}

func withLogLineGroup(group string, work func() error) error {
	if !runningInGitHubActions() {
		printHeading(group)
		return work()
	}

	fmt.Println("::group::" + group)

	err := work()

	fmt.Println("::endgroup::")

	return err
}

// FIXME: lineSplitterTee should maybe go to gokit? other use is in Varasto

type lineSplitterTee struct {
	buf           []byte // buffer before receiving \n
	lineCompleted func(string)
	mu            sync.Mutex
}

// returns io.Writer that tees full lines to lineCompleted callback
func newLineSplitterTee(sink io.Writer, lineCompleted func(string)) io.Writer {
	return io.MultiWriter(sink, &lineSplitterTee{
		buf:           []byte{},
		lineCompleted: lineCompleted,
	})
}

func (l *lineSplitterTee) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf = append(l.buf, data...)

	// as long as we have lines, chop the buffer down
	for {
		idx := strings.IndexByte(string(l.buf), '\n')
		if idx == -1 {
			break
		}

		l.lineCompleted(string(l.buf[0:idx]))

		l.buf = l.buf[idx+1:]
	}

	return len(data), nil
}

// tech debt: can't update to newer Go to use this func from gokit
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	} else {
		return b
	}
}

// write image tags as Markdown to GitHub actions's workflow summary so it's easy from their UI to spot
// which images were published as result of the build.
func githubStepSummaryWriteImages(stepSummaryFilename string, images []imageBuildOutput) error {
	withErr := func(err error) error { return fmt.Errorf("githubStepSummaryWriteImages: %w", err) }

	stepSummaryFile, err := os.OpenFile(stepSummaryFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return withErr(err)
	}
	defer stepSummaryFile.Close()

	return githubStepSummaryWriteImagesWithWriter(stepSummaryFile, images)
}

func githubStepSummaryWriteImagesWithWriter(stepSummaryFile io.Writer, images []imageBuildOutput) error {
	lines := []string{}
	for _, image := range images {
		parsed := dockertag.Parse(image.tag)
		if parsed == nil {
			return fmt.Errorf("failed to parse docker tag: %s", image.tag)
		}

		// "fn61/varasto" => "varasto"
		imageBasename := path.Base(parsed.Repository)

		lines = append(lines, "## Image: "+imageBasename, "", "```", image.tag, "```", "", "")
	}

	if _, err := stepSummaryFile.Write([]byte(strings.Join(lines, "\n"))); err != nil {
		return err
	}

	return nil
}
