package audio

import (
	"context"
	"errors"
	"time"

	"github.com/gen2brain/malgo"
)

var ErrDeviceNotFound = errors.New("audio input device not found")
var ErrAudioQueueFull = errors.New("audio input queue is full")

type Device struct {
	id   malgo.DeviceID
	Name string
}

type Chunk struct {
	Samples    []float32
	SampleRate uint32
}

type Source interface {
	Devices() ([]Device, error)

	Run(
		ctx context.Context,
		device Device,
		consume func(context.Context, Chunk) error,
	) error

	Close() error
}

type Config struct {
	SampleRate uint32
	Latency    time.Duration
	QueueSize  int
}
