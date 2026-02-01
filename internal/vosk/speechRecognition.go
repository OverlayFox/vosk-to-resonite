package vosk

import (
	"encoding/json"
	"regexp"
	"strings"

	vosk "github.com/alphacep/vosk-api/go"
	"github.com/rodaine/numwords"
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
	// Enable word-level recognition for better accuracy
	rec.SetWords(1)

	return &Vosk{
		logger:     log.With().Str("component", "vosk").Logger(),
		modelPath:  modelPath,
		recognizer: rec,
	}, nil
}

// AcceptAudio processes audio data and returns final results only
func (v *Vosk) AcceptAudio(data []byte) string {
	state := v.recognizer.AcceptWaveform(data)

	// Only process final results (when state returns 1)
	if state == 1 {
		var result map[string]any
		jsonStr := v.recognizer.Result()
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return ""
		}

		// logger.Debug().Interface("result", result).Msg("Final recognition result")

		var parsedText string
		if text, ok := result["text"].(string); ok && text != "" {
			parsedText = numwords.ParseString(text)
		}

		var re = regexp.MustCompile(`(?i)(grow|shrink)(?:\W+(?:\w+\W+){0,6}?)(\d+(?:\s+point\s+\d+|\.\d+)?)`)
		matches := re.FindAllStringSubmatch(parsedText, -1)

		for _, match := range matches {
			trigger := match[1]
			numberRaw := match[2]

			cleanNumber := strings.ReplaceAll(numberRaw, " point ", ".")
			logger.Debug().Str("trigger", trigger).Str("numberRaw", numberRaw).Str("cleanNumber", cleanNumber).Msg("Parsed command")
		}
	}
	return ""
}
