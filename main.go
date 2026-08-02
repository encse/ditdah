package main

import (
	"context"
	"fmt"
	"morsemanual/internal/audio"
	"morsemanual/internal/decoder"
	"path/filepath"
)

func main() {
	s, err := audio.NewInput(audio.DefaultConfig())
	if err != nil {
		panic(err)
	}
	devices, err := s.Devices()
	if err != nil {
		panic(err)
	}

	for _, device := range devices {
		println(device.Name)
	}

	ctx, cancel := context.WithCancel(context.Background())

	weightsDir := filepath.Join(".", "morse.weights")

	d, err := decoder.NewStreaming(weightsDir)
	if err != nil {
		panic(err)
	}

	s.Run(
		ctx,
		devices[1],
		func(ctx context.Context, chunk audio.Chunk) error {
			return d.Process(
				ctx,
				chunk,
				func(ctx context.Context, token decoder.AudioToken) error {
					if token != decoder.CTCBlank {
						fmt.Printf("%s ", token)
					}
					return nil
				},
			)
		},
	)

	cancel()

	// root := "."
	// weightsDir := filepath.Join(root, "morse.weights")
	// testDir := filepath.Join(root, "morse.onnx.testdata")

	// projectionIndices := []int{0, 2, 4, 6}
	// projection := make([]linearLayer, 0, len(projectionIndices))
	// inputSize := featureSize
	// for _, index := range projectionIndices {
	// 	prefix := fmt.Sprintf("frame_projection__%d", index)
	// 	projection = append(projection, linearLayer{
	// 		inputSize:  inputSize,
	// 		outputSize: hiddenSize,
	// 		weight:     mustLoad(filepath.Join(weightsDir, prefix+"__weight.f32"), hiddenSize*inputSize),
	// 		bias:       mustLoad(filepath.Join(weightsDir, prefix+"__bias.f32"), hiddenSize),
	// 	})
	// 	inputSize = hiddenSize
	// }

	// classifier := linearLayer{
	// 	inputSize:  hiddenSize,
	// 	outputSize: tokenCount,
	// 	weight:     mustLoad(filepath.Join(weightsDir, "classifier__weight.f32"), tokenCount*hiddenSize),
	// 	bias:       mustLoad(filepath.Join(weightsDir, "classifier__bias.f32"), tokenCount),
	// }

	// weightIH := mustLoad(filepath.Join(weightsDir, "sequence_encoder__weight_ih_l0.f32"), 4*hiddenSize*hiddenSize)
	// weightHH := mustLoad(filepath.Join(weightsDir, "sequence_encoder__weight_hh_l0.f32"), 4*hiddenSize*hiddenSize)
	// biasIH := mustLoad(filepath.Join(weightsDir, "sequence_encoder__bias_ih_l0.f32"), 4*hiddenSize)
	// biasHH := mustLoad(filepath.Join(weightsDir, "sequence_encoder__bias_hh_l0.f32"), 4*hiddenSize)

	// features := mustLoad(filepath.Join(testDir, "features.f32"), frames*featureSize)
	// hidden := mustLoad(filepath.Join(testDir, "hidden_state.f32"), hiddenSize)
	// cell := mustLoad(filepath.Join(testDir, "cell_state.f32"), hiddenSize)

	// logits := make([]float32, frames*tokenCount)
	// bufferA := make([]float32, hiddenSize)
	// bufferB := make([]float32, hiddenSize)
	// gates := make([]float32, 4*hiddenSize)

	// for frame := 0; frame < frames; frame++ {
	// 	current := features[frame*featureSize : (frame+1)*featureSize]
	// 	for layerIndex, layer := range projection {
	// 		output := bufferA
	// 		if layerIndex%2 == 1 {
	// 			output = bufferB
	// 		}
	// 		layer.forward(current, output)
	// 		relu(output)
	// 		current = output
	// 	}

	// 	lstmStep(current, hidden, cell, weightIH, weightHH, biasIH, biasHH, gates)
	// 	classifier.forward(hidden, logits[frame*tokenCount:(frame+1)*tokenCount])
	// }

	// expectedLogits := mustLoad(filepath.Join(testDir, "expected_logits.f32"), frames*tokenCount)
	// expectedHidden := mustLoad(filepath.Join(testDir, "expected_hidden_state.f32"), hiddenSize)
	// expectedCell := mustLoad(filepath.Join(testDir, "expected_cell_state.f32"), hiddenSize)

	// compare("logits", expectedLogits, logits)
	// compare("hidden_state", expectedHidden, hidden)
	// compare("cell_state", expectedCell, cell)

	// matches := 0
	// fmt.Print("argmax path: ")
	// for frame := 0; frame < frames; frame++ {
	// 	expectedToken := argmax(expectedLogits[frame*tokenCount : (frame+1)*tokenCount])
	// 	actualToken := argmax(logits[frame*tokenCount : (frame+1)*tokenCount])
	// 	if expectedToken == actualToken {
	// 		matches++
	// 	}
	// 	fmt.Printf("%d", actualToken)
	// 	if frame+1 != frames {
	// 		fmt.Print(" ")
	// 	}
	// }
	// fmt.Println()
	// fmt.Printf("argmax_agreement=%d/%d (%.2f%%)\n", matches, frames, 100*float64(matches)/frames)
}
