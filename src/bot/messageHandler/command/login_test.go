package command

import (
	"strings"
	"testing"
)

func TestGenerateLoginCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		code, err := generateLoginCode()
		if (err != nil) {
			t.Fatal(err)
		}

		if (len(code) != loginCodeLength) {
			t.Fatalf("code %q has %d characters, want %d", code, len(code), loginCodeLength)
		}

		for _, character := range code {
			if (!strings.ContainsRune(loginCodeAlphabet, character)) {
				t.Fatalf("code %q uses a character outside the alphabet", code)
			}
		}

		if (seen[code]) {
			t.Fatalf("code %q repeated within 1000 draws", code)
		}
		seen[code] = true
	}
}
