package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gen2brain/malgo"
)

type input struct {
	context *malgo.AllocatedContext
	config  Config
}

func DefaultConfig() Config {
	return Config{
		SampleRate: 8000,
		Latency:    100 * time.Millisecond,
		QueueSize:  32,
	}
}

func (c Config) validate() error {
	if c.SampleRate == 0 {
		return fmt.Errorf("sample rate must be positive")
	}
	if c.Latency <= 0 {
		return fmt.Errorf("latency must be positive")
	}
	if c.QueueSize <= 0 {
		return fmt.Errorf("queue size must be positive")
	}
	return nil
}

func NewInput(config Config) (Source, error) {
	err := config.validate()

	if err != nil {
		return nil, err
	}

	malgoContext, err := malgo.InitContext(
		nil,
		malgo.ContextConfig{},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize audio context: %w", err)
	}

	return &input{
		context: malgoContext,
		config:  config,
	}, nil
}

func (i *input) Close() error {
	if i.context == nil {
		return nil
	}

	err := i.context.Uninit()
	i.context.Free()
	i.context = nil

	if err != nil {
		return fmt.Errorf("uninitialize audio context: %w", err)
	}

	return nil
}

// Devices returns the currently available audio capture devices.
func (i *input) Devices() ([]Device, error) {
	infos, err := i.context.Context.Devices(malgo.Capture)
	if err != nil {
		return nil, fmt.Errorf("list audio input devices: %w", err)
	}

	devices := make([]Device, 0, len(infos))
	for _, info := range infos {
		devices = append(devices, Device{
			id:   info.ID,
			Name: info.Name(),
		})
	}

	return devices, nil
}

func (i *input) Run(
	ctx context.Context,
	device Device,
	consume func(context.Context, Chunk) error,
) error {

	if consume == nil {
		return errors.New("audio consumer is nil")
	}

	config := malgo.DefaultDeviceConfig(malgo.Capture)
	config.Capture.DeviceID = device.id.Pointer()
	config.Capture.Format = malgo.FormatF32
	config.Capture.Channels = 1

	config.SampleRate = i.config.SampleRate

	config.PeriodSizeInMilliseconds = uint32(
		math.Max(1, math.Round(i.config.Latency.Seconds()*1000)),
	)

	chunks := make(chan []float32, i.config.QueueSize)
	callbackErrors := make(chan error, 1)

	callbacks := malgo.DeviceCallbacks{
		Data: func(
			_ []byte,
			inputSamples []byte,
			frameCount uint32,
		) {
			expectedBytes := int(frameCount) * 4
			if len(inputSamples) < expectedBytes {
				reportCallbackError(
					callbackErrors,
					fmt.Errorf(
						"audio callback returned %d bytes for %d frames",
						len(inputSamples),
						frameCount,
					),
				)
				return
			}

			samples := decodeFloat32(inputSamples[:expectedBytes])

			select {
			case chunks <- samples:
			default:
				reportCallbackError(callbackErrors, ErrAudioQueueFull)
			}
		},
	}

	capture, err := malgo.InitDevice(
		i.context.Context,
		config,
		callbacks,
	)

	if err != nil {
		return fmt.Errorf("initialize audio input device: %w", err)
	}
	defer capture.Uninit()

	if err := capture.Start(); err != nil {
		return fmt.Errorf("start audio input device: %w", err)
	}

	sampleRate := capture.CaptureInternalSampleRate()
	if sampleRate == 0 {
		return errors.New("audio input device reported zero sample rate")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-callbackErrors:
			return err

		case samples := <-chunks:
			chunk := Chunk{
				Samples:    samples,
				SampleRate: sampleRate,
			}

			if err := consume(ctx, chunk); err != nil {
				return fmt.Errorf("consume audio chunk: %w", err)
			}
		}
	}
}

func decodeFloat32(data []byte) []float32 {
	samples := make([]float32, len(data)/4)

	for index := range samples {
		bits := binary.LittleEndian.Uint32(data[index*4:])
		samples[index] = math.Float32frombits(bits)
	}

	return samples
}

func reportCallbackError(destination chan<- error, err error) {
	select {
	case destination <- err:
	default:
	}
}
