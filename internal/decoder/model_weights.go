package decoder

import "embed"

//go:embed weights/*.f32
var modelWeights embed.FS
