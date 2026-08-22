//go:build arm64 && goexperiment.simd

package model

import (
	"ditdah/internal/decoder/model/core"
	"ditdah/internal/decoder/model/simdmodel"
)

func LoadModel() (core.Model, error) {
	return simdmodel.LoadSIMDModel()
}
