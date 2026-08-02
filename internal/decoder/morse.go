package decoder

var morseDecodingTable = map[string]string{
	".-":   "A",
	"-...": "B",
	"-.-.": "C",
	"-..":  "D",
	".":    "E",
	"..-.": "F",
	"--.":  "G",
	"....": "H",
	"..":   "I",
	".---": "J",
	"-.-":  "K",
	".-..": "L",
	"--":   "M",
	"-.":   "N",
	"---":  "O",
	".--.": "P",
	"--.-": "Q",
	".-.":  "R",
	"...":  "S",
	"-":    "T",
	"..-":  "U",
	"...-": "V",
	".--":  "W",
	"-..-": "X",
	"-.--": "Y",
	"--..": "Z",

	"-----": "0",
	".----": "1",
	"..---": "2",
	"...--": "3",
	"....-": "4",
	".....": "5",
	"-....": "6",
	"--...": "7",
	"---..": "8",
	"----.": "9",

	".-.-.-":  ".",
	"--..--":  ",",
	"..--..":  "?",
	".----.":  "'",
	"-.-.--":  "!",
	"-..-.":   "/",
	"-.--.":   "(",
	"-.--.-":  ")",
	".-...":   "&",
	"---...":  ":",
	"-.-.-.":  ";",
	"-...-":   "=",
	".-.-.":   "+",
	"-....-":  "-",
	"..--.-":  "_",
	".-..-.":  "\"",
	"...-..-": "$",
	".--.-.":  "@",
}

func decodeMorseTokens(tokens []AudioToken) string {
	morse := make([]byte, 0, len(tokens))

	for _, token := range tokens {
		switch token {
		case Dit:
			morse = append(morse, '.')

		case Dah:
			morse = append(morse, '-')
		}
	}

	if len(morse) == 0 {
		return ""
	}

	code := string(morse)

	if character, found := morseDecodingTable[code]; found {
		return character
	}

	return "[" + code + "]"
}
