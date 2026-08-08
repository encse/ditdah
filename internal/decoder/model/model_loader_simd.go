//go:build arm64 && goexperiment.simd

package model

import (
	"fmt"
	"morsemanual/internal/decoder/model/core"
	"morsemanual/internal/decoder/model/simdmodel"
)

func LoadModel() (core.Model, error) {
	fmt.Println("Loading SIMD model")
	return simdmodel.LoadSIMDModel()
}
