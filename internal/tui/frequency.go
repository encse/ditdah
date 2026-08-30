package tui

import (
	"fmt"
	"strings"
)

// FormatFrequencyMHz formats an exact hertz value in MHz with at least three
// fractional digits and only the additional digits required for precision.
func FormatFrequencyMHz(frequencyHz uint64) string {
	megahertz := frequencyHz / 1_000_000
	fraction := fmt.Sprintf("%06d", frequencyHz%1_000_000)
	for len(fraction) > 3 && strings.HasSuffix(fraction, "0") {
		fraction = fraction[:len(fraction)-1]
	}
	return fmt.Sprintf("%d.%s MHz", megahertz, fraction)
}
