//go:build arm64 && goexperiment.simd

package model

import (
	"morsemanual/internal/decoder/model/core"
	"morsemanual/internal/decoder/model/simdmodel"
)

func LoadModel() (core.Model, error) {
	return simdmodel.LoadSIMDModel()
}
