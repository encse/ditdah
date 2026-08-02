package decoder

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

const (
	referenceDirectory = "model/reference"
	logitTolerance     = 5e-5
)

type tensorMetadata struct {
	File         string `json:"file"`
	Shape        []int  `json:"shape"`
	ElementCount int    `json:"element_count"`
	DType        string `json:"dtype"`
	ByteOrder    string `json:"byte_order"`
}

type referenceMetadata struct {
	SampleRate int `json:"sample_rate"`

	Input struct {
		Waveform tensorMetadata `json:"waveform"`
	} `json:"input"`

	ExpectedOutputs struct {
		Logits tensorMetadata `json:"logits"`
	} `json:"expected_outputs"`
}

func TestInferenceMatchesPython(t *testing.T) {
	metadata := loadReferenceMetadata(
		t,
		filepath.Join(referenceDirectory, "metadata.json"),
	)

	if metadata.SampleRate != 8_000 {
		t.Fatalf(
			"unexpected sample rate: got %d, want 8000",
			metadata.SampleRate,
		)
	}

	waveformMetadata := metadata.Input.Waveform
	logitsMetadata := metadata.ExpectedOutputs.Logits

	validateTensorMetadata(
		t,
		"waveform",
		waveformMetadata,
		1,
	)

	validateTensorMetadata(
		t,
		"logits",
		logitsMetadata,
		2,
	)

	if waveformMetadata.DType != "float32" {
		t.Fatalf(
			"unexpected waveform dtype: got %q, want float32",
			waveformMetadata.DType,
		)
	}

	if logitsMetadata.DType != "float32" {
		t.Fatalf(
			"unexpected logits dtype: got %q, want float32",
			logitsMetadata.DType,
		)
	}

	if waveformMetadata.ByteOrder != "little-endian" {
		t.Fatalf(
			"unexpected waveform byte order: got %q, want little-endian",
			waveformMetadata.ByteOrder,
		)
	}

	if logitsMetadata.ByteOrder != "little-endian" {
		t.Fatalf(
			"unexpected logits byte order: got %q, want little-endian",
			logitsMetadata.ByteOrder,
		)
	}

	frameCount := logitsMetadata.Shape[0]
	referenceTokenCount := logitsMetadata.Shape[1]

	if referenceTokenCount != tokenCount {
		t.Fatalf(
			"unexpected token count: got %d, want %d",
			referenceTokenCount,
			tokenCount,
		)
	}

	if waveformMetadata.ElementCount != waveformMetadata.Shape[0] {
		t.Fatalf(
			"waveform metadata mismatch: element_count=%d shape=%v",
			waveformMetadata.ElementCount,
			waveformMetadata.Shape,
		)
	}

	if logitsMetadata.ElementCount != frameCount*referenceTokenCount {
		t.Fatalf(
			"logits metadata mismatch: element_count=%d shape=%v",
			logitsMetadata.ElementCount,
			logitsMetadata.Shape,
		)
	}

	waveform := loadReferenceFloat32(
		t,
		filepath.Join(
			referenceDirectory,
			waveformMetadata.File,
		),
		waveformMetadata.ElementCount,
	)

	expectedLogits := loadReferenceFloat32(
		t,
		filepath.Join(
			referenceDirectory,
			logitsMetadata.File,
		),
		logitsMetadata.ElementCount,
	)

	if len(waveform)%frameSamples != 0 {
		t.Fatalf(
			"waveform length %d is not divisible by frame size %d",
			len(waveform),
			frameSamples,
		)
	}

	actualFrameCount := len(waveform) / frameSamples

	if actualFrameCount != frameCount {
		t.Fatalf(
			"frame count mismatch: waveform produces %d frames, metadata says %d",
			actualFrameCount,
			frameCount,
		)
	}

	model, err := LoadModel()
	if err != nil {
		t.Fatalf("load model: %v", err)
	}

	transform := NewSTFT()

	actualLogits := make(
		[]float32,
		0,
		frameCount*referenceTokenCount,
	)

	for frameIndex := 0; frameIndex < frameCount; frameIndex++ {
		start := frameIndex * frameSamples
		end := start + frameSamples

		features, err := transform.Compute(
			waveform[start:end],
		)
		if err != nil {
			t.Fatalf(
				"compute STFT for frame %d: %v",
				frameIndex,
				err,
			)
		}

		logits, err := model.Step(features)
		if err != nil {
			t.Fatalf(
				"run model for frame %d: %v",
				frameIndex,
				err,
			)
		}

		actualLogits = append(actualLogits, logits...)
	}

	assertFloat32Close(
		t,
		expectedLogits,
		actualLogits,
		logitTolerance,
	)

	assertArgmaxPathEqual(
		t,
		expectedLogits,
		actualLogits,
		referenceTokenCount,
	)
}

func validateTensorMetadata(
	t *testing.T,
	name string,
	metadata tensorMetadata,
	expectedDimensions int,
) {
	t.Helper()

	if metadata.File == "" {
		t.Fatalf("%s metadata has no file name", name)
	}

	if len(metadata.Shape) != expectedDimensions {
		t.Fatalf(
			"unexpected %s shape: got %v, want %d dimensions",
			name,
			metadata.Shape,
			expectedDimensions,
		)
	}

	if metadata.ElementCount <= 0 {
		t.Fatalf(
			"%s element count must be positive, got %d",
			name,
			metadata.ElementCount,
		)
	}
}

func loadReferenceMetadata(
	t *testing.T,
	path string,
) referenceMetadata {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read metadata %q: %v",
			path,
			err,
		)
	}

	var metadata referenceMetadata

	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf(
			"decode metadata %q: %v",
			path,
			err,
		)
	}

	return metadata
}

func loadReferenceFloat32(
	t *testing.T,
	path string,
	expectedCount int,
) []float32 {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read reference tensor %q: %v",
			path,
			err,
		)
	}

	expectedBytes := expectedCount * 4

	if len(data) != expectedBytes {
		t.Fatalf(
			"%s: got %d bytes, want %d",
			path,
			len(data),
			expectedBytes,
		)
	}

	values := make([]float32, expectedCount)

	for index := range values {
		offset := index * 4

		bits := binary.LittleEndian.Uint32(
			data[offset : offset+4],
		)

		values[index] = math.Float32frombits(bits)
	}

	return values
}

func assertFloat32Close(
	t *testing.T,
	expected []float32,
	actual []float32,
	tolerance float64,
) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf(
			"logits length mismatch: got %d, want %d",
			len(actual),
			len(expected),
		)
	}

	var maximumDifference float64
	var differenceSum float64
	maximumIndex := 0

	for index := range expected {
		difference := math.Abs(
			float64(expected[index] - actual[index]),
		)

		differenceSum += difference

		if difference > maximumDifference {
			maximumDifference = difference
			maximumIndex = index
		}
	}

	meanDifference := differenceSum / float64(len(expected))

	t.Logf(
		"logits: max_abs_error=%g mean_abs_error=%g",
		maximumDifference,
		meanDifference,
	)

	if maximumDifference > tolerance {
		t.Fatalf(
			"logits differ at index %d: max difference %g exceeds tolerance %g; expected=%g actual=%g",
			maximumIndex,
			maximumDifference,
			tolerance,
			expected[maximumIndex],
			actual[maximumIndex],
		)
	}
}

func assertArgmaxPathEqual(
	t *testing.T,
	expected []float32,
	actual []float32,
	tokenCount int,
) {
	t.Helper()

	frameCount := len(expected) / tokenCount

	for frameIndex := 0; frameIndex < frameCount; frameIndex++ {
		start := frameIndex * tokenCount
		end := start + tokenCount

		expectedToken := AudioToken(
			argmax(expected[start:end]),
		)

		actualToken := AudioToken(
			argmax(actual[start:end]),
		)

		if expectedToken != actualToken {
			t.Fatalf(
				"frame %d token mismatch: got %s, want %s",
				frameIndex,
				actualToken,
				expectedToken,
			)
		}
	}
}
