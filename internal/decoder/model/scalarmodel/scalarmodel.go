package scalarmodel

import (
	"fmt"
	"math"
	"morsemanual/internal/decoder/model/artifacts"
	modelpkg "morsemanual/internal/decoder/model/core"
)

type model struct {
	projection [4]modelpkg.LinearLayer
	classifier modelpkg.LinearLayer

	weightIH []float32
	weightHH []float32
	biasIH   []float32
	biasHH   []float32

	hidden []float32
	cell   []float32
	gates  []float32

	bufferA []float32
	bufferB []float32
	logits  []float32
}

func LoadScalarModel() (modelpkg.Model, error) {
	projectionIndices := [...]int{0, 2, 4, 6}

	model := &model{
		hidden:  make([]float32, modelpkg.HiddenSize),
		cell:    make([]float32, modelpkg.HiddenSize),
		gates:   make([]float32, 4*modelpkg.HiddenSize),
		bufferA: make([]float32, modelpkg.HiddenSize),
		bufferB: make([]float32, modelpkg.HiddenSize),
		logits:  make([]float32, modelpkg.TokenCount),
	}

	inputSize := modelpkg.FeatureSize

	for layerIndex, weightIndex := range projectionIndices {
		prefix := fmt.Sprintf("frame_projection__%d", weightIndex)

		weight, err := artifacts.LoadFloat32(
			prefix+"__weight.f32",
			modelpkg.HiddenSize*inputSize,
		)
		if err != nil {
			return nil, err
		}

		bias, err := artifacts.LoadFloat32(
			prefix+"__bias.f32",
			modelpkg.HiddenSize,
		)
		if err != nil {
			return nil, err
		}

		model.projection[layerIndex] = modelpkg.LinearLayer{
			InputSize:  inputSize,
			OutputSize: modelpkg.HiddenSize,
			Weight:     weight,
			Bias:       bias,
		}

		inputSize = modelpkg.HiddenSize
	}

	classifierWeight, err := artifacts.LoadFloat32(
		"classifier__weight.f32",
		modelpkg.TokenCount*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	classifierBias, err := artifacts.LoadFloat32(
		"classifier__bias.f32",
		modelpkg.TokenCount,
	)
	if err != nil {
		return nil, err
	}

	model.classifier = modelpkg.LinearLayer{
		InputSize:  modelpkg.HiddenSize,
		OutputSize: modelpkg.TokenCount,
		Weight:     classifierWeight,
		Bias:       classifierBias,
	}

	model.weightIH, err = artifacts.LoadFloat32(
		"sequence_encoder__weight_ih_l0.f32",
		4*modelpkg.HiddenSize*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	model.weightHH, err = artifacts.LoadFloat32(
		"sequence_encoder__weight_hh_l0.f32",
		4*modelpkg.HiddenSize*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	model.biasIH, err = artifacts.LoadFloat32(
		"sequence_encoder__bias_ih_l0.f32",
		4*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	model.biasHH, err = artifacts.LoadFloat32(
		"sequence_encoder__bias_hh_l0.f32",
		4*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	return model, nil
}

// Step processes one 65-element spectrogram feature frame.
//
// The returned slice belongs to the model and is reused by the next call.
func (m *model) Step(features []float32) ([]float32, error) {
	if len(features) != modelpkg.FeatureSize {
		return nil, fmt.Errorf(
			"expected %d features, got %d",
			modelpkg.FeatureSize,
			len(features),
		)
	}

	current := features

	for layerIndex := range m.projection {
		output := m.bufferA

		if layerIndex%2 == 1 {
			output = m.bufferB
		}

		m.projection[layerIndex].Forward(current, output)
		relu(output)
		current = output
	}

	m.lstmStep(current)
	m.classifier.Forward(m.hidden, m.logits)

	return m.logits, nil
}

func (m *model) Reset() {
	clear(m.hidden)
	clear(m.cell)
	clear(m.gates)
	clear(m.bufferA)
	clear(m.bufferB)
	clear(m.logits)
}

func (m *model) lstmStep(input []float32) {
	gateRows := 4 * modelpkg.HiddenSize

	for row := 0; row < gateRows; row++ {
		sum := m.biasIH[row] + m.biasHH[row]

		inputOffset := row * modelpkg.HiddenSize
		hiddenOffset := row * modelpkg.HiddenSize

		for column := 0; column < modelpkg.HiddenSize; column++ {
			sum += m.weightIH[inputOffset+column] * input[column]
			sum += m.weightHH[hiddenOffset+column] * m.hidden[column]
		}

		m.gates[row] = sum
	}

	// PyTorch gate order: input, forget, cell candidate, output.
	for index := 0; index < modelpkg.HiddenSize; index++ {
		inputGate := sigmoid(m.gates[index])
		forgetGate := sigmoid(m.gates[modelpkg.HiddenSize+index])
		candidate := float32(
			math.Tanh(
				float64(m.gates[2*modelpkg.HiddenSize+index]),
			),
		)
		outputGate := sigmoid(m.gates[3*modelpkg.HiddenSize+index])

		nextCell := forgetGate*m.cell[index] + inputGate*candidate
		nextHidden := outputGate * float32(
			math.Tanh(float64(nextCell)),
		)

		m.cell[index] = nextCell
		m.hidden[index] = nextHidden
	}
}

func relu(values []float32) {
	for index, value := range values {
		if value < 0 {
			values[index] = 0
		}
	}
}

func sigmoid(value float32) float32 {
	return 1 / (1 + float32(math.Exp(-float64(value))))
}
