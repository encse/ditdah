//go:build arm64 && goexperiment.simd

package simdmodel_test

import (
	"math/rand"
	"morsemanual/internal/decoder/model/scalarmodel"
	"morsemanual/internal/decoder/model/simdmodel"
	"testing"
)

func makeBenchmarkFloats(size int) []float32 {
	rng := rand.New(rand.NewSource(42))

	values := make([]float32, size)
	for i := range values {
		values[i] = rng.Float32()*2 - 1
	}

	return values
}

func BenchmarkModelStep(b *testing.B) {
	features := makeBenchmarkFloats(65)
	b.Run("scalar", func(b *testing.B) {
		model, err := scalarmodel.LoadScalarModel()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()

		for b.Loop() {
			if _, err := model.Step(features); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("simd", func(b *testing.B) {
		model, err := simdmodel.LoadSIMDModel()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()

		for b.Loop() {
			if _, err := model.Step(features); err != nil {
				b.Fatal(err)
			}
		}
	})
}
