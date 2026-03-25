package main

import (
	"context"
	"os"

	"github.com/arykalin/whisper-cli/internal/app"
	"github.com/rs/zerolog/log"
)

func main() {
	application := app.NewDefault()
	if err := application.Run(context.Background(), os.Args[1:]); err != nil {
		log.Fatal().Err(err).Msg("execution failed")
	}
}
