//go:build !arm64 || !goexperiment.simd

package model

import (
	"ditdah/internal/decoder/model/core"
	"ditdah/internal/decoder/model/scalarmodel"
)

func LoadModel() (core.Model, error) {
	return scalarmodel.LoadScalarModel()
}
