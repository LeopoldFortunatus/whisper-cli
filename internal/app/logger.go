package app

import (
	"log/syslog"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func initLogger() zerolog.Logger {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}

	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "whisper-cli")
	if err != nil {
		logger := zerolog.New(consoleWriter).With().Timestamp().Logger()
		log.Logger = logger
		log.Warn().Err(err).Msg("syslog unavailable, using stderr")
		return logger
	}

	logger := zerolog.New(
		zerolog.MultiLevelWriter(consoleWriter, zerolog.SyslogLevelWriter(writer)),
	).With().Timestamp().Logger()
	log.Logger = logger
	return logger
}
