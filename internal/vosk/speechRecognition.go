package vosk

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/OverlayFox/vosk-to-resonite/internal/resonite"
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
func (v *Vosk) AcceptAudio(data []byte) []resonite.Command {
	state := v.recognizer.AcceptWaveform(data)

	// Only process final results (when state returns 1)
	commands := make([]resonite.Command, 0)
	if state == 1 {
		var result map[string]any
		jsonStr := v.recognizer.Result()
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil
		}

		logger.Debug().Interface("result", result).Msg("Final recognition result")

		var parsedText string
		if text, ok := result["text"].(string); ok && text != "" {
			parsedText = numwords.ParseString(text)
		}

		triggers := []string{string(resonite.CommandTypeGrow), string(resonite.CommandTypeShrink)}
		triggerPattern := fmt.Sprintf(`(?i)\b(%s)\b`, strings.Join(triggers, "|"))
		reTrigger := regexp.MustCompile(triggerPattern)

		names := []string{"neo", "overlay", "bronze", "loki"}
		namePattern := fmt.Sprintf(`(?i)\b(%s)\b`, strings.Join(names, "|"))
		reName := regexp.MustCompile(namePattern)

		reNumber := regexp.MustCompile(`(?i)(\d+(?:\s+point\s+\d+|\.\d+)?)`)

		triggerMatches := reTrigger.FindAllStringIndex(parsedText, -1)

		for _, tLoc := range triggerMatches {
			triggerStart, triggerEnd := tLoc[0], tLoc[1]
			triggerWord := parsedText[triggerStart:triggerEnd]

			textAfter := parsedText[triggerEnd:]
			numMatch := reNumber.FindStringIndex(textAfter)

			var extractedNum float64
			if numMatch != nil {
				gapText := textAfter[:numMatch[0]]
				wordCount := len(strings.Fields(gapText))

				if wordCount <= 6 {
					rawNum := textAfter[numMatch[0]:numMatch[1]]
					numberStr := strings.ReplaceAll(rawNum, " point ", ".")
					num, err := strconv.ParseFloat(numberStr, 64)
					if err == nil {
						extractedNum = num
					} else {
						v.logger.Error().Err(err).Str("rawNum", rawNum).Str("numberStr", numberStr).Msg("Failed to parse extracted number")
					}
				}
			}

			nameMatches := reName.FindAllStringIndex(parsedText, -1)
			closestName := ""
			shortestDitance := math.MaxInt64

			for _, nLoc := range nameMatches {
				nameStart, nameEnd := nLoc[0], nLoc[1]
				nameStr := parsedText[nameStart:nameEnd]

				var dist int
				if nameEnd < triggerStart {
					dist = triggerStart - nameEnd // name is before trigger
				} else if nameStart > triggerEnd {
					dist = nameStart - triggerEnd // name is after trigger
				} else {
					dist = 0 // overlapping, should not happen
				}

				if dist < shortestDitance {
					shortestDitance = dist
					closestName = nameStr
				}
			}

			command := resonite.Command{
				Type:  resonite.CommandType(strings.ToLower(triggerWord)),
				Value: extractedNum,
				Name:  strings.ToLower(closestName),
			}

			commands = append(commands, command)
			logger.Info().Interface("command", command).Msg("Extracted command from speech")
		}
	}

	return commands
}
