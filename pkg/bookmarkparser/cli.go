package bookmarkparser

import (
	"context"
	"log"
	"os"

	"github.com/function61/gokit/app/cli"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "bookmarks-build",
		Short: "Build code bookmarks database from current directory",
		Args:  cobra.NoArgs,
		Run: cli.RunnerNoArgs(func(ctx context.Context, _ *log.Logger) error {
			return ParseBookmarks(ctx, ".", os.Stdout)
		}),
	}
}
