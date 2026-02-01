package vosk

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/OverlayFox/vosk-to-resonite/internal/resonite"
	vosk "github.com/alphacep/vosk-api/go"
	"github.com/rodaine/numwords"
	"github.com/rs/zerolog"
)

const (
	MaxDistanceNumberToTrigger = 6 // max number of words between trigger and number
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
		v.logger.Debug().Msg("Processing final recognition result")

		var result map[string]any
		jsonStr := v.recognizer.Result()
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil
		}

		var parsedText string
		if text, ok := result["text"].(string); ok && text != "" {
			parsedText = numwords.ParseString(text)
		}
		v.logger.Debug().Str("recognized_text", parsedText).Msg("Final recognized speech")

		// define triggers
		triggerPattern := fmt.Sprintf(`(?i)\b(%s)\b`, strings.Join(resonite.StringToCommandTypeList, "|"))
		reTrigger := regexp.MustCompile(triggerPattern)

		// define number pattern
		var safeExprs []string
		for _, expr := range resonite.Expressions {
			safeExprs = append(safeExprs, fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(expr)))
		}
		customPattern := strings.Join(safeExprs, "|")
		finalPattern := fmt.Sprintf(`(?i)(%s|\d+(?:\s+point\s+\d+|\.\d+)?)`, customPattern)
		reNumber := regexp.MustCompile(finalPattern)

		triggerMatches := reTrigger.FindAllStringIndex(parsedText, -1)
		for _, tLoc := range triggerMatches {
			triggerStart, triggerEnd := tLoc[0], tLoc[1]
			triggerWord := resonite.StringToCommandType(parsedText[triggerStart:triggerEnd])
			if triggerWord == resonite.CommandTypeUndefined {
				continue
			}

			textAfter := parsedText[triggerEnd:]
			numMatch := reNumber.FindStringIndex(textAfter)

			var extractedNum float64
			var extractedUnit resonite.CommandUnit
			if numMatch != nil {
				gapText := textAfter[:numMatch[0]]
				wordCount := len(strings.Fields(gapText))

				if wordCount <= MaxDistanceNumberToTrigger {
					rawNum := textAfter[numMatch[0]:numMatch[1]]
					numberStr := strings.ReplaceAll(rawNum, " point ", ".")
					num, err := strconv.ParseFloat(numberStr, 64)
					if err == nil {
						extractedNum = num
					} else {
						var ok bool
						extractedNum, ok = resonite.ExpressionToPercent(rawNum)
						extractedUnit = resonite.CommandUnitPercent
						if !ok {
							v.logger.Warn().Str("number_str", rawNum).Msg("Failed to parse number from recognized speech")
							continue
						}
					}

					// extract the word after the number as unit
					wordsAfterNumber := strings.Fields(textAfter[numMatch[1]:])
					if len(wordsAfterNumber) > 0 {
						extractedUnit = resonite.StringToCommandUnit(wordsAfterNumber[0])
					}
				}
			}

			command := resonite.Command{
				Type:  triggerWord,
				Value: extractedNum,
				Unit:  extractedUnit,
			}
			commands = append(commands, command)
		}
	}

	return commands
}
