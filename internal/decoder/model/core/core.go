// Package core contains the contracts and dimensions shared by model
// implementations.
package core

type Model interface {
	Step(features []float32) ([]float32, error)
	Reset()
}

const (
	FeatureSize = 65
	HiddenSize  = 256
	TokenCount  = 5
)

type LinearLayer struct {
	InputSize  int
	OutputSize int
	Weight     []float32
	Bias       []float32
}

func (l *LinearLayer) Forward(input, output []float32) {
	for row := 0; row < l.OutputSize; row++ {
		sum := l.Bias[row]
		offset := row * l.InputSize

		for column := 0; column < l.InputSize; column++ {
			sum += l.Weight[offset+column] * input[column]
		}

		output[row] = sum
	}
}

func Argmax(values []float32) int {
	bestIndex := 0
	bestValue := values[0]

	for index := 1; index < len(values); index++ {
		if values[index] > bestValue {
			bestValue = values[index]
			bestIndex = index
		}
	}

	return bestIndex
}
