package decoder

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	featureSize = 65
	hiddenSize  = 256
	tokenCount  = 5
)

type linearLayer struct {
	inputSize  int
	outputSize int
	weight     []float32
	bias       []float32
}

func (l *linearLayer) forward(input, output []float32) {
	for row := 0; row < l.outputSize; row++ {
		sum := l.bias[row]
		offset := row * l.inputSize

		for column := 0; column < l.inputSize; column++ {
			sum += l.weight[offset+column] * input[column]
		}

		output[row] = sum
	}
}

type model struct {
	projection [4]linearLayer
	classifier linearLayer

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

type Model interface {
	Step(features []float32) ([]float32, error)
	Reset()
}

func LoadModel() (Model, error) {
	projectionIndices := [...]int{0, 2, 4, 6}

	model := &model{
		hidden:  make([]float32, hiddenSize),
		cell:    make([]float32, hiddenSize),
		gates:   make([]float32, 4*hiddenSize),
		bufferA: make([]float32, hiddenSize),
		bufferB: make([]float32, hiddenSize),
		logits:  make([]float32, tokenCount),
	}

	inputSize := featureSize

	for layerIndex, weightIndex := range projectionIndices {
		prefix := fmt.Sprintf("frame_projection__%d", weightIndex)

		weight, err := loadFloat32(
			prefix+"__weight.f32",
			hiddenSize*inputSize,
		)
		if err != nil {
			return nil, err
		}

		bias, err := loadFloat32(
			prefix+"__bias.f32",
			hiddenSize,
		)
		if err != nil {
			return nil, err
		}

		model.projection[layerIndex] = linearLayer{
			inputSize:  inputSize,
			outputSize: hiddenSize,
			weight:     weight,
			bias:       bias,
		}

		inputSize = hiddenSize
	}

	classifierWeight, err := loadFloat32(
		"classifier__weight.f32",
		tokenCount*hiddenSize,
	)
	if err != nil {
		return nil, err
	}

	classifierBias, err := loadFloat32(
		"classifier__bias.f32",
		tokenCount,
	)
	if err != nil {
		return nil, err
	}

	model.classifier = linearLayer{
		inputSize:  hiddenSize,
		outputSize: tokenCount,
		weight:     classifierWeight,
		bias:       classifierBias,
	}

	model.weightIH, err = loadFloat32(
		"sequence_encoder__weight_ih_l0.f32",
		4*hiddenSize*hiddenSize,
	)
	if err != nil {
		return nil, err
	}

	model.weightHH, err = loadFloat32(
		"sequence_encoder__weight_hh_l0.f32",
		4*hiddenSize*hiddenSize,
	)
	if err != nil {
		return nil, err
	}

	model.biasIH, err = loadFloat32(
		"sequence_encoder__bias_ih_l0.f32",
		4*hiddenSize,
	)
	if err != nil {
		return nil, err
	}

	model.biasHH, err = loadFloat32(
		"sequence_encoder__bias_hh_l0.f32",
		4*hiddenSize,
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
	if len(features) != featureSize {
		return nil, fmt.Errorf(
			"expected %d features, got %d",
			featureSize,
			len(features),
		)
	}

	current := features

	for layerIndex := range m.projection {
		output := m.bufferA

		if layerIndex%2 == 1 {
			output = m.bufferB
		}

		m.projection[layerIndex].forward(current, output)
		relu(output)
		current = output
	}

	m.lstmStep(current)
	m.classifier.forward(m.hidden, m.logits)

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
	gateRows := 4 * hiddenSize

	for row := 0; row < gateRows; row++ {
		sum := m.biasIH[row] + m.biasHH[row]

		inputOffset := row * hiddenSize
		hiddenOffset := row * hiddenSize

		for column := 0; column < hiddenSize; column++ {
			sum += m.weightIH[inputOffset+column] * input[column]
			sum += m.weightHH[hiddenOffset+column] * m.hidden[column]
		}

		m.gates[row] = sum
	}

	// PyTorch gate order: input, forget, cell candidate, output.
	for index := 0; index < hiddenSize; index++ {
		inputGate := sigmoid(m.gates[index])
		forgetGate := sigmoid(m.gates[hiddenSize+index])
		candidate := float32(
			math.Tanh(
				float64(m.gates[2*hiddenSize+index]),
			),
		)
		outputGate := sigmoid(m.gates[3*hiddenSize+index])

		nextCell := forgetGate*m.cell[index] + inputGate*candidate
		nextHidden := outputGate * float32(
			math.Tanh(float64(nextCell)),
		)

		m.cell[index] = nextCell
		m.hidden[index] = nextHidden
	}
}

func loadFloat32(name string, count int) ([]float32, error) {
	path := "model/weights/" + name

	data, err := modelWeights.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read embedded model tensor %q: %w", path, err)
	}

	expectedBytes := count * 4
	if len(data) != expectedBytes {
		return nil, fmt.Errorf(
			"%s: got %d bytes, expected %d",
			path,
			len(data),
			expectedBytes,
		)
	}

	values := make([]float32, count)

	for index := range values {
		offset := index * 4
		bits := binary.LittleEndian.Uint32(data[offset : offset+4])
		values[index] = math.Float32frombits(bits)
	}

	return values, nil
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
