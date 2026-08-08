package decoder

import "testing"

func TestDecoderOffsets(t *testing.T) {
	want := [decoderCount]int{0, 53, 80}
	if decoderOffsets != want {
		t.Fatalf("decoderOffsets = %v, want %v", decoderOffsets, want)
	}
}

func TestMajorityToken(t *testing.T) {
	tests := []struct {
		name   string
		tokens [decoderCount]AudioToken
		want   AudioToken
	}{
		{"first two agree", [decoderCount]AudioToken{Dit, Dit, Dah}, Dit},
		{"last two agree", [decoderCount]AudioToken{Dit, Dah, Dah}, Dah},
		{"first and last agree", [decoderCount]AudioToken{EndWord, Dit, EndWord}, EndWord},
		{"all agree", [decoderCount]AudioToken{EndCharacter, EndCharacter, EndCharacter}, EndCharacter},
		{"no majority prefers blank", [decoderCount]AudioToken{Dit, Dah, CTCBlank}, CTCBlank},
		{"no majority and no blank uses middle", [decoderCount]AudioToken{Dit, Dah, EndCharacter}, Dah},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := majorityToken(test.tokens); got != test.want {
				t.Fatalf("majorityToken(%v) = %v, want %v", test.tokens, got, test.want)
			}
		})
	}
}
