package main

// Trigger is a CLI-based UI for running arbitrary commands (think: compile or test or both) from
// user-defined event sources. (I have mapped a foot pedal as the trigger fire)
//
// If you want bob to (fast-)compile when the trigger fires (works from inside a dev container too!):
//
//     $ bob trigger 'bob build --fast'
//
// You can then fire the trigger from the host system by running:
//
//     $ bob trigger fire
//
// Only one "trigger server" can be active at at time. (If needed, named triggers could be added.)

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/function61/gokit/log/logex"
	"github.com/function61/gokit/net/netutil"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/gokit/sync/taskrunner"
	"github.com/spf13/cobra"
)

const (
	// taking advantage of the fact that /tmp/build is bind-mounted to dev containers (for caching)
	triggerSockPath = "/tmp/build/trigger.sock"
)

func triggerEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger [commandToRunWhenTriggered]",
		Short: `Like "watch", but with user-defined event source to run cmds (compile/test/anything)`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()

			osutil.ExitIfError(trigger(
				osutil.CancelOnInterruptOrTerminate(rootLogger),
				args[0],
				rootLogger))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "fire",
		Short: "Fire the trigger, usually from event source in the host system",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(triggerFire(
				osutil.CancelOnInterruptOrTerminate(nil)))
		},
	})

	return cmd
}

func triggerFire(ctx context.Context) error {
	// we could use something sophisticated, even a HTTP server, but we use a cheap man's version
	// where each connection open is the trigger fire. no real data is transmitted in either direction

	client, err := (&net.Dialer{}).DialContext(ctx, "unix", triggerSockPath)
	if err != nil {
		return err
	}

	return client.Close()
}

// pro-tip: you need to inside the host:
//
//	$ chgrp $(id -g) /tmp/build
func trigger(ctx context.Context, cmd string, logger *log.Logger) error {
	// this channel gets a signal each time we should activate the trigger
	triggerFireReq := make(chan void, 1)

	tasks := taskrunner.New(ctx, logger)

	// the "UI" for the trigger command. we'll spend most of our time waiting for trigger to fire,
	// and when it does, we'll run the trigger's target command and display its output + exit code
	tasks.Start("cmdrunner", func(ctx context.Context) error {
		var runningCommand *exec.Cmd
		runningCommandExited := make(chan error, 1)
		var stopRunningCommand context.CancelFunc

		start := func() {
			var subCtx context.Context
			subCtx, stopRunningCommand = context.WithCancel(ctx)

			runningCommand = exec.CommandContext(subCtx, "sh", "-c", cmd)
			runningCommand.Stdin = os.Stdin
			runningCommand.Stdout = os.Stdout
			runningCommand.Stderr = os.Stderr

			go func() {
				runningCommandExited <- runningCommand.Run()
			}()
		}

		handleStopped := func(err error) {
			// show easy-to-read status line for its exit. we've just shown stdout/stderr above this
			if err != nil {
				fmt.Fprintf(os.Stderr, "╰╴╴ ✗: %s\n", err.Error())
			} else {
				fmt.Fprintf(os.Stderr, "╰╴╴ ✓\n")
			}

			runningCommand = nil
		}

		stopIfRunning := func() {
			if runningCommand != nil {
				// TODO: graceful stop instead of forced kill
				stopRunningCommand()

				handleStopped(<-runningCommandExited)
			}
		}

		for {
			select {
			case <-ctx.Done():
				stopIfRunning()

				return nil
			case err := <-runningCommandExited: // spontaneous stop
				handleStopped(err)
			case <-triggerFireReq:
				stopIfRunning()

				start()
			}
		}
	})

	tasks.Start("trigger-server", func(ctx context.Context) error {
		return triggerServerDeliverIncomingTriggerFires(ctx, triggerFireReq)
	})

	return tasks.Wait()
}

func triggerServerDeliverIncomingTriggerFires(ctx context.Context, triggerFireReq chan<- void) error {
	defer close(triggerFireReq)

	return netutil.ListenUnixAllowOwnerAndGroup(ctx, triggerSockPath, func(listener net.Listener) error {
		for {
			client, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return nil // context canceled
				} else {
					return err // actual error
				}
			}

			triggerFireReq <- void{}

			_ = client.Close() // cleanup, conn open only used as a trigger signal
		}
	})
}

type void struct{}
