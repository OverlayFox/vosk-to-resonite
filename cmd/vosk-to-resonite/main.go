package main

import (
	"os"

	"github.com/OverlayFox/vosk-to-resonite/internal/vosk"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	err := vosk.GetModel(logger)
	if err != nil {
		panic(err)
	}
}
