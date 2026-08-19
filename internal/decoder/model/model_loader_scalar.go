//go:build !arm64 || !goexperiment.simd

package model

import (
	"morsemanual/internal/decoder/model/core"
	"morsemanual/internal/decoder/model/scalarmodel"
)

func LoadModel() (core.Model, error) {
	return scalarmodel.LoadScalarModel()
}
