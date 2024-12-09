package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/function61/gokit/app/cli"
	"github.com/spf13/cobra"
)

func smokeTestEntrypoint() *cobra.Command {
	longRunning := false

	cmd := &cobra.Command{
		Use: "smoketest",
		// Short: "Code bookmarks management",
		Args: cobra.NoArgs,
		Run: cli.RunnerNoArgs(func(ctx context.Context, _ *log.Logger) error {
			if longRunning {
				return smokeTest(ctx, smokeTestDefinition{
					imageRef: "smoketest-longrunning",
					kind:     "stay-up",
				})
			} else {
				return smokeTest(ctx, smokeTestDefinition{
					imageRef: "smoketest",
					args:     []string{"--help"},
					kind:     "exit-immediately",
					exitImmediatelyOptions: exitImmediatelyOptions{
						outputContains: "Usage: curl [options...] <url>",
					},
				})
			}
		}),
	}

	cmd.Flags().BoolVarP(&longRunning, "long-running", "l", longRunning, "Test a long-running daemon")

	return cmd
}

type smokeTestDefinition struct {
	imageRef               string
	args                   []string
	kind                   string // "exit-immediately" | "stay-up"
	exitImmediatelyOptions exitImmediatelyOptions
}

type exitImmediatelyOptions struct {
	outputContains string
}

func smokeTest(ctx context.Context, testSpec smokeTestDefinition) error {
	withErr := func(err error) error { return fmt.Errorf("smokeTest: %w", err) }

	// TODO: generate
	const containerName = "smoketest-123"

	// kill signal to stop container as fast as possible when we want to stop.
	cmd := []string{"docker", "run", "--rm", "-t", "--stop-signal=SIGKILL", "--name=" + containerName, testSpec.imageRef}
	cmd = append(cmd, testSpec.args...)

	smokeTest := exec.CommandContext(ctx, cmd[0], cmd[1:]...)

	switch testSpec.kind {
	case "exit-immediately":
		return smokeTestExitImmediately(smokeTest, testSpec)
	case "stay-up":
		childCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return smokeTestStayUp(childCtx, smokeTest, containerName)
	default:
		return withErr(fmt.Errorf("unknown kind: %s", testSpec.kind))
	}
}

func smokeTestExitImmediately(smokeTest *exec.Cmd, testSpec smokeTestDefinition) error {
	withErr := func(err error) error { return fmt.Errorf("smokeTestExitImmediately: %w", err) }

	output, err := smokeTest.CombinedOutput()
	if err != nil {
		return withErr(fmt.Errorf("%w; output: %s", err, string(output)))
	}

	if outputContains := testSpec.exitImmediatelyOptions.outputContains; outputContains != "" {
		if !strings.Contains(string(output), outputContains) {
			return withErr(fmt.Errorf("output should contain '%s' but did not. output was: %s", outputContains, string(output)))
		}
	}

	return nil
}

func smokeTestStayUp(ctx context.Context, smokeTest *exec.Cmd, containerName string) error {
	withErr := func(err error) error { return fmt.Errorf("smokeTestStayUp: %w", err) }

	containerExited := make(chan error, 1)
	go func() {
		containerExited <- smokeTest.Run()
	}()

	select {
	case err := <-containerExited: // program exited itself
		return withErr(fmt.Errorf("unexpected container exit: %w", err))
	case <-ctx.Done(): // stop requested before container exited by itself
		// ask the container to stop. for some reason sending SIGINT to `$ docker` will not work
		// (neither from shell, maybe it just forwards signals to the container?) but instead we must use `$ docker stop`.
		//
		// NOTE: using background context because we can't derive context from already-canceled context.
		if output, err := exec.CommandContext(context.Background(), "docker", "stop", containerName).CombinedOutput(); err != nil {
			// probably a race - program exited at the same time?
			return withErr(fmt.Errorf("got context cancel but stopping: %w: %s", err, string(output)))
		}

		<-containerExited

		if reason := ctx.Err(); errors.Is(reason, context.DeadlineExceeded) {
			// happy path: container stays up for at least the timeout that we wanted
			return nil
		} else { // maybe user cancelled the wait
			return withErr(reason)
		}
	}
}
