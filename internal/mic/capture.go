package mic

import (
	"github.com/gen2brain/malgo"
	"github.com/rs/zerolog"
)

type Microphone struct {
	logger zerolog.Logger

	deviceConfig malgo.DeviceConfig

	ctx *malgo.AllocatedContext
}

func NewMicrophone(log zerolog.Logger) (*Microphone, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		// log.Debug().Msg(msg)
	})
	if err != nil {
		return nil, err
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 16000
	deviceConfig.Alsa.NoMMap = 1

	return &Microphone{
		logger: log.With().Str("component", "microphone").Logger(),

		deviceConfig: deviceConfig,

		ctx: ctx,
	}, nil
}

func (m *Microphone) ListCaptureDevices() ([]malgo.DeviceInfo, error) {
	return m.ctx.Devices(malgo.Capture)
}

func (m *Microphone) StartCapture(deviceInfo malgo.DeviceInfo) (<-chan []byte, error) {
	m.deviceConfig.Capture.DeviceID = deviceInfo.ID.Pointer()

	recvChan := make(chan []byte, 1024)
	onRecvFrames := func(_, input []byte, framecount uint32) {
		data := make([]byte, len(input))
		copy(data, input)
		recvChan <- data
	}

	captureCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	device, err := malgo.InitDevice(m.ctx.Context, m.deviceConfig, captureCallbacks)
	if err != nil {
		return nil, err
	}

	m.logger.Info().Str("device", deviceInfo.Name()).Msg("Starting audio capture on device")
	return recvChan, device.Start()
}
