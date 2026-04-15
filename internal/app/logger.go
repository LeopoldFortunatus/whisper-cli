package app

import (
	"log/syslog"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func initLogger() zerolog.Logger {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}

	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "whisper-cli")
	if err != nil {
		return zerolog.New(consoleWriter).With().Timestamp().Logger()
	}

	logger := zerolog.New(
		zerolog.MultiLevelWriter(consoleWriter, zerolog.SyslogLevelWriter(writer)),
	).With().Timestamp().Logger()
	return logger
}
