package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/arykalin/whisper-cli/internal/app"
	"github.com/arykalin/whisper-cli/internal/completion"
	"github.com/arykalin/whisper-cli/internal/config"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		log.Fatal().Err(err).Msg("execution failed")
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if handled, err := runCompletion(args, stdout); handled {
		return err
	}

	application := app.NewDefault()
	if err := application.Run(ctx, args); err != nil {
		var helpErr *config.HelpError
		if errors.As(err, &helpErr) {
			_, writeErr := fmt.Fprint(stdout, helpErr.Usage)
			return writeErr
		}
		return err
	}
	return nil
}

func runCompletion(args []string, stdout io.Writer) (bool, error) {
	if len(args) == 0 || args[0] != "completion" {
		return false, nil
	}
	if len(args) != 2 || args[1] != "bash" {
		return true, errors.New("usage: whisper-cli completion bash")
	}

	_, err := io.WriteString(stdout, completion.BashScript())
	return true, err
}
