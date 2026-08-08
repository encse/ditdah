//go:build arm64 && goexperiment.simd

package simdmodel

import (
	"math"
	"math/rand"
	"morsemanual/internal/decoder/model/core"
	"morsemanual/internal/decoder/model/scalarmodel"
	"testing"
)

func TestSIMDModelMatchesScalar(t *testing.T) {
	scalarModel, err := scalarmodel.LoadScalarModel()
	if err != nil {
		t.Fatal(err)
	}

	simdModel, err := LoadSIMDModel()
	if err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewSource(42))

	const (
		steps        = 1000
		absTolerance = 1e-4
	)

	var maxAbsError float64
	var maxErrorStep int
	var maxErrorLogit int

	for step := 0; step < steps; step++ {
		features := make([]float32, core.FeatureSize)

		for i := range features {
			features[i] = rng.Float32()*2 - 1
		}

		scalarLogits, err := scalarModel.Step(features)
		if err != nil {
			t.Fatal(err)
		}

		simdLogits, err := simdModel.Step(features)
		if err != nil {
			t.Fatal(err)
		}

		for i := range scalarLogits {
			err := math.Abs(float64(
				scalarLogits[i] - simdLogits[i],
			))

			if err > maxAbsError {
				maxAbsError = err
				maxErrorStep = step
				maxErrorLogit = i
			}

			if err > absTolerance {
				t.Fatalf(
					"step=%d logit=%d scalar=%g simd=%g abs_error=%g",
					step,
					i,
					scalarLogits[i],
					simdLogits[i],
					err,
				)
			}
		}

		scalarToken := core.Argmax(scalarLogits)
		simdToken := core.Argmax(simdLogits)

		if scalarToken != simdToken {
			t.Fatalf(
				"step=%d token mismatch: scalar=%d simd=%d\nscalar=%v\nsimd=%v",
				step,
				scalarToken,
				simdToken,
				scalarLogits,
				simdLogits,
			)
		}
	}

	t.Logf(
		"max_abs_error=%g at step=%d logit=%d",
		maxAbsError,
		maxErrorStep,
		maxErrorLogit,
	)

}
