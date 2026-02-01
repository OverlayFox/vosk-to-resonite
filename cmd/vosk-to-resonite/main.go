package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OverlayFox/vosk-to-resonite/internal/mic"
	"github.com/OverlayFox/vosk-to-resonite/internal/vosk"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	_, err := vosk.NewVosk(logger) // replace with proper model path if needed
	if err != nil {
		panic(err)
	}

	microphone, err := mic.NewMicrophone(logger)
	if err != nil {
		panic(err)
	}

	devices, err := microphone.ListCaptureDevices()
	if err != nil {
		panic(err)
	}

	// Display available devices
	fmt.Println("Available audio input devices:")
	for i, device := range devices {
		fmt.Printf("[%d] %s\n", i, device.Name())
	}

	// Prompt user to select
	fmt.Print("\nSelect device number: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selectedIndex, err := strconv.Atoi(input)
	if err != nil || selectedIndex < 0 || selectedIndex >= len(devices) {
		logger.Fatal().Msg("Invalid device selection")
	}

	logger.Info().Str("name", devices[selectedIndex].Name()).Msg("Selected device")

	micChan, err := microphone.StartCapture(devices[selectedIndex])
	if err != nil {
		panic(err)
	}

	for {
		data := <-micChan
		_ = data
		//logger.Debug().Bytes("data", data).Msg("Captured audio frame")
	}
}
