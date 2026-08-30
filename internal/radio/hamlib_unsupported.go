//go:build !darwin || !cgo

package radio

import (
	"context"
	"errors"
)

type unsupportedService struct{}

// New returns a service that reports the unavailable native Hamlib build.
func New() Service { return unsupportedService{} }

func (unsupportedService) Models() ([]Model, error) {
	return nil, errors.New("Hamlib is not available on this platform yet")
}

func (unsupportedService) Ports() ([]string, error) {
	return nil, errors.New("Hamlib is not available on this platform yet")
}

func (unsupportedService) Check(context.Context, Config) (uint64, error) {
	return 0, errors.New("Hamlib is not available on this platform yet")
}
