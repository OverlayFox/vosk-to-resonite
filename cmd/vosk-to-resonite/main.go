package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OverlayFox/vosk-to-resonite/internal/mic"
	"github.com/OverlayFox/vosk-to-resonite/internal/vosk"
	"github.com/rodaine/numwords"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	// Initialize Vosk and Microphone
	voskInstance, err := vosk.NewVosk(logger)
	if err != nil {
		panic(err)
	}
	microphone, err := mic.NewMicrophone(logger)
	if err != nil {
		panic(err)
	}

	// List available audio input devices
	devices, err := microphone.ListCaptureDevices()
	if err != nil {
		panic(err)
	}

	fmt.Println("Available audio input devices:")
	for i, device := range devices {
		fmt.Printf("[%d] %s\n", i, device.Name())
	}

	fmt.Print("\nSelect device number: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selectedIndex, err := strconv.Atoi(input)
	if err != nil || selectedIndex < 0 || selectedIndex >= len(devices) {
		logger.Fatal().Msg("Invalid device selection")
	}
	logger.Info().Str("name", devices[selectedIndex].Name()).Msg("Selected device")

	// Start capturing audio from the selected device
	micChan, err := microphone.StartCapture(devices[selectedIndex])
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 0, 8192) // 512ms buffer at 16kHz 16-bit mono
	for {
		data := <-micChan
		buffer = append(buffer, data...)

		// Process when we have enough data (about 512ms worth)
		if len(buffer) >= 8192 {
			result := voskInstance.AcceptAudio(buffer)
			if result != "" {
				splitResult := strings.Split(result, " ")
				for _, word := range splitResult {
					if num, err := numwords.ParseInt(word); err == nil {
						logger.Info().Str("original", word).Int("number", num).Msg("Converted words to number")
					} else {
						logger.Info().Str("word", word).Msg("Non-numeric word recognized")
					}
				}
			}
			buffer = buffer[:0]
		}
	}
}
