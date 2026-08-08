//go:build !arm64 || !goexperiment.simd

package model

import (
	"fmt"
	"morsemanual/internal/decoder/model/core"
	"morsemanual/internal/decoder/model/scalarmodel"
)

func LoadModel() (core.Model, error) {
	fmt.Println("Loading Scalar model")
	return scalarmodel.LoadScalarModel()
}
