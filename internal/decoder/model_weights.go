package decoder

import "embed"

//go:embed model/weights/*.f32
var modelWeights embed.FS
