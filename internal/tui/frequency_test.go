package tui

import "testing"

func TestFormatFrequencyMHzUsesOnlyNecessaryPrecision(t *testing.T) {
	tests := map[uint64]string{
		28_039_000: "28.039 MHz",
		28_039_600: "28.0396 MHz",
		28_039_601: "28.039601 MHz",
	}
	for frequency, want := range tests {
		if got := FormatFrequencyMHz(frequency); got != want {
			t.Errorf("FormatFrequencyMHz(%d) = %q, want %q", frequency, got, want)
		}
	}
}
