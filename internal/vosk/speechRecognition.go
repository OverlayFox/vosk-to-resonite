package vosk

import (
	vosk "github.com/alphacep/vosk-api/go"
	"github.com/rs/zerolog"
)

type Vosk struct {
	logger zerolog.Logger

	modelPath  string
	recognizer *vosk.VoskRecognizer
}

func NewVosk(log zerolog.Logger) (*Vosk, error) {
	modelPath, err := getModel(log)
	if err != nil {
		panic(err)
	}

	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return nil, err
	}
	rec, err := vosk.NewRecognizer(model, 16000.0)
	if err != nil {
		return nil, err
	}

	return &Vosk{
		logger:     log.With().Str("component", "vosk").Logger(),
		modelPath:  modelPath,
		recognizer: rec,
	}, nil
}
