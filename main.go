package main

import (
	"context"
	"io"
	"os"

	"github.com/arykalin/whisper-cli/internal/app"
	"github.com/arykalin/whisper-cli/internal/cli"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, app.NewDefault()); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer, application *app.Application) error {
	return cli.Run(ctx, application, args, stdout, stderr)
}
