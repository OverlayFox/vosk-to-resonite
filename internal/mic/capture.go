package mic

import (
	"github.com/gen2brain/malgo"
	"github.com/rs/zerolog"
)

type Microphone struct {
	logger zerolog.Logger

	streamChan chan []byte

	deviceConfig malgo.DeviceConfig
	device       *malgo.Device

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

	m.streamChan = make(chan []byte, 1024)
	onRecvFrames := func(_, input []byte, framecount uint32) {
		data := make([]byte, len(input))
		copy(data, input)
		m.streamChan <- data
	}

	captureCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	var err error
	m.device, err = malgo.InitDevice(m.ctx.Context, m.deviceConfig, captureCallbacks)
	if err != nil {
		return nil, err
	}

	m.logger.Info().Str("device", deviceInfo.Name()).Msg("Starting audio capture on device")
	return m.streamChan, m.device.Start()
}

func (m *Microphone) Close() {
	m.logger.Info().Msg("Stopping audio capture and releasing resources")

	if m.device != nil {
		m.device.Uninit()
		m.device = nil
	}

	if m.streamChan != nil {
		close(m.streamChan)
		m.streamChan = nil
	}

	if m.ctx != nil {
		m.ctx.Free()
		m.ctx = nil
	}

	m.logger.Info().Msg("Microphone resources released")
}
