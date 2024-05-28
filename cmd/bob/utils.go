package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func passthroughStdoutAndStderr(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

func isEnvVarPresent(key string) bool {
	return os.Getenv(key) != ""
}

func allDevShellCommands(devShellCommands []DevShellCommand) []string {
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
