package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

func NewLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}

	return zerolog.New(consoleWriter).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		Str("service", "stringAnalyzer").
		Str("environment", "development").
		Logger().With().Stack().Logger()

}
