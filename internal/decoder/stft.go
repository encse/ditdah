package decoder

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	fftSize       = 160
	frequencyBins = 65
)

type stft struct {
	fft          *fourier.FFT
	window       []float64
	fftInput     []float64
	coefficients []complex128
	features     []float32
	coherentGain float64
}

type STFT interface {
	Compute(samples []float32) ([]float32, error)
}

func NewSTFT() STFT {
	window := make([]float64, fftSize)

	var windowSum float64
	for index := range window {
		// Equivalent to torch.hann_window(160, periodic=true).
		window[index] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(index)/float64(fftSize)))
		windowSum += window[index]
	}

	return &stft{
		fft:          fourier.NewFFT(fftSize),
		window:       window,
		fftInput:     make([]float64, fftSize),
		coefficients: make([]complex128, fftSize/2+1),
		features:     make([]float32, frequencyBins),
		coherentGain: windowSum / 2.0,
	}
}

// Compute computes the exact feature scale used by the Python model.
func (s *stft) Compute(samples []float32) ([]float32, error) {
	if len(samples) != fftSize {
		return nil, fmt.Errorf(
			"expected %d audio samples, got %d",
			fftSize,
			len(samples),
		)
	}

	for index, sample := range samples {
		s.fftInput[index] = float64(sample) * s.window[index]
	}

	s.coefficients = s.fft.Coefficients(
		s.coefficients,
		s.fftInput,
	)

	for index := 0; index < frequencyBins; index++ {
		coefficient := s.coefficients[index]

		magnitude := math.Hypot(
			real(coefficient),
			imag(coefficient),
		) / s.coherentGain

		s.features[index] = float32(magnitude * magnitude)
	}

	return s.features, nil
}
