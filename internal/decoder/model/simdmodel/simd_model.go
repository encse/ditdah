//go:build arm64 && goexperiment.simd

package simdmodel

import (
	"fmt"
	"math"
	"ditdah/internal/decoder/model/artifacts"
	modelpkg "ditdah/internal/decoder/model/core"
	"simd/archsimd"
)

func forwardLinear(l *modelpkg.LinearLayer, input, output []float32) {
	for row := 0; row < l.OutputSize; row++ {
		offset := row * l.InputSize

		output[row] = l.Bias[row] + dotFloat32SIMD(
			l.Weight[offset:offset+l.InputSize],
			input,
		)
	}
}

type simdModel struct {
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

func LoadSIMDModel() (modelpkg.Model, error) {
	projectionIndices := [...]int{0, 2, 4, 6}

	m := &simdModel{
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

		m.projection[layerIndex] = modelpkg.LinearLayer{
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

	m.classifier = modelpkg.LinearLayer{
		InputSize:  modelpkg.HiddenSize,
		OutputSize: modelpkg.TokenCount,
		Weight:     classifierWeight,
		Bias:       classifierBias,
	}

	m.weightIH, err = artifacts.LoadFloat32(
		"sequence_encoder__weight_ih_l0.f32",
		4*modelpkg.HiddenSize*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	m.weightHH, err = artifacts.LoadFloat32(
		"sequence_encoder__weight_hh_l0.f32",
		4*modelpkg.HiddenSize*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	m.biasIH, err = artifacts.LoadFloat32(
		"sequence_encoder__bias_ih_l0.f32",
		4*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	m.biasHH, err = artifacts.LoadFloat32(
		"sequence_encoder__bias_hh_l0.f32",
		4*modelpkg.HiddenSize,
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *simdModel) Step(features []float32) ([]float32, error) {
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

		forwardLinear(&m.projection[layerIndex], current, output)
		reluSIMD(output)

		current = output
	}

	m.lstmStep(current)
	forwardLinear(&m.classifier, m.hidden, m.logits)

	return m.logits, nil
}

func (m *simdModel) Reset() {
	clear(m.hidden)
	clear(m.cell)
	clear(m.gates)
	clear(m.bufferA)
	clear(m.bufferB)
	clear(m.logits)
}

func (m *simdModel) lstmStep(input []float32) {
	gateRows := 4 * modelpkg.HiddenSize

	for row := 0; row < gateRows; row++ {
		offset := row * modelpkg.HiddenSize

		sum := m.biasIH[row] + m.biasHH[row]

		sum += dotFloat32SIMD(
			m.weightIH[offset:offset+modelpkg.HiddenSize],
			input,
		)

		sum += dotFloat32SIMD(
			m.weightHH[offset:offset+modelpkg.HiddenSize],
			m.hidden,
		)

		m.gates[row] = sum
	}

	// PyTorch gate order: input, forget, cell candidate, output.
	for index := 0; index < modelpkg.HiddenSize; index++ {
		inputGate := sigmoid(m.gates[index])

		forgetGate := sigmoid(
			m.gates[modelpkg.HiddenSize+index],
		)

		candidate := float32(
			math.Tanh(
				float64(m.gates[2*modelpkg.HiddenSize+index]),
			),
		)

		outputGate := sigmoid(
			m.gates[3*modelpkg.HiddenSize+index],
		)

		nextCell :=
			forgetGate*m.cell[index] +
				inputGate*candidate

		nextHidden :=
			outputGate *
				float32(math.Tanh(float64(nextCell)))

		m.cell[index] = nextCell
		m.hidden[index] = nextHidden
	}
}

func dotFloat32SIMD(a, b []float32) float32 {
	var acc0 archsimd.Float32x4
	var acc1 archsimd.Float32x4
	var acc2 archsimd.Float32x4
	var acc3 archsimd.Float32x4

	i := 0

	// Four independent accumulators reduce the FMA dependency chain.
	for ; i+16 <= len(a); i += 16 {
		a0 := archsimd.LoadFloat32x4(a[i:])
		b0 := archsimd.LoadFloat32x4(b[i:])
		acc0 = a0.MulAdd(b0, acc0)

		a1 := archsimd.LoadFloat32x4(a[i+4:])
		b1 := archsimd.LoadFloat32x4(b[i+4:])
		acc1 = a1.MulAdd(b1, acc1)

		a2 := archsimd.LoadFloat32x4(a[i+8:])
		b2 := archsimd.LoadFloat32x4(b[i+8:])
		acc2 = a2.MulAdd(b2, acc2)

		a3 := archsimd.LoadFloat32x4(a[i+12:])
		b3 := archsimd.LoadFloat32x4(b[i+12:])
		acc3 = a3.MulAdd(b3, acc3)
	}

	acc := acc0.Add(acc1)
	acc = acc.Add(acc2)
	acc = acc.Add(acc3)

	var lanes [4]float32
	acc.Store(lanes[:])

	sum := lanes[0] +
		lanes[1] +
		lanes[2] +
		lanes[3]

	for ; i < len(a); i++ {
		sum += a[i] * b[i]
	}

	return sum
}

func reluSIMD(values []float32) {
	zero := archsimd.BroadcastFloat32x4(0)

	i := 0

	for ; i+4 <= len(values); i += 4 {
		v := archsimd.LoadFloat32x4(values[i:])
		v = v.Max(zero)
		v.Store(values[i:])
	}

	for ; i < len(values); i++ {
		if values[i] < 0 {
			values[i] = 0
		}
	}
}

func sigmoid(value float32) float32 {
	return 1 / (1 + float32(math.Exp(-float64(value))))
}
