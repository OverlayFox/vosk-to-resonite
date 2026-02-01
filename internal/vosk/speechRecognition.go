package vosk

import (
	"encoding/json"
	"math"
	"strconv"
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

		var splitResult []string
		if text, ok := result["text"].(string); ok && text != "" {
			parsedText := numwords.ParseString(text)
			logger.Debug().Str("parsed", parsedText).Msg("Parsed text")
			splitResult = strings.Split(parsedText, " ")
		}
		if len(splitResult) <= 0 {
			return ""
		}

		// Extract numbers, handling decimals
		index := 0
		isDecimal := false
		numberList := map[int][]int{}
		for _, word := range splitResult {
			num, err := strconv.Atoi(word)
			if err == nil {
				numberList[index] = append(numberList[index], num)
				if !isDecimal {
					index++
				}
				continue
			}
			if word == "point" {
				isDecimal = true
				index--
			} else {
				if isDecimal {
					index++
				}
				isDecimal = false
			}
		}

		// Combine integer and decimal parts
		var finalNumbers []float64
		for _, nums := range numberList {
			if len(nums) == 1 {
				finalNumbers = append(finalNumbers, float64(nums[0]))
				continue
			}
			intPart := nums[0]
			decimalPart := 0.0
			for i, n := range nums[1:] {
				decimalPart += float64(n) / math.Pow10(i+1)
			}
			finalNumbers = append(finalNumbers, float64(intPart)+decimalPart)
		}

		logger.Info().Interface("finalNumbers", finalNumbers).Msg("Final Numbers")
	}

	return ""
}
