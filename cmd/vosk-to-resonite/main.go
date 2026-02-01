package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/OverlayFox/vosk-to-resonite/internal/mic"
	"github.com/OverlayFox/vosk-to-resonite/internal/resonite"
	"github.com/OverlayFox/vosk-to-resonite/internal/vosk"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	// Initialize Vosk and Microphone
	voskInstance, err := vosk.NewVosk(logger)
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to initialize Vosk")
	}
	defer voskInstance.Close()

	// Microphone Setup
	microphone, err := mic.NewMicrophone(logger)
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to initialize microphone")
	}
	defer microphone.Close()

	// Start Resonite WebSocket server
	resoniteWebSocket, err := resonite.NewWebSocketServer(logger, resonite.WebSocketServerConfig{Port: 8080})
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to start Resonite WebSocket server")
	}
	resoniteWebSocket.Start()
	defer resoniteWebSocket.Close()

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
		logger.Panic().Err(err).Msg("Failed to start audio capture")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	buffer := make([]byte, 0, 8192) // 512ms buffer at 16kHz 16-bit mono
	for {
		select {
		case data := <-micChan:
			buffer = append(buffer, data...)

			// Process when we have enough data (about 512ms worth)
			if len(buffer) >= 8192 {
				commands := voskInstance.AcceptAudio(buffer)
				if len(commands) > 0 {
					for _, cmd := range commands {
						resoniteWebSocket.Write(cmd)
					}
				}
				buffer = buffer[:0]
			}
		case sig := <-quit:
			logger.Info().Str("signal", sig.String()).Msg("Received termination signal, exiting...")
			return
		}
	}
}
