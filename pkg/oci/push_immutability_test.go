package oci

import "testing"

func TestImmutableAlpha2Tag(t *testing.T) {
	tests := []struct {
		tag       string
		version   string
		immutable bool
	}{
		{"1.2.3-alpha.1", "1.2.3-alpha.1", true},
		{"1.2.3-beta.3", "1.2.3-beta.3", true},
		{"1.2.3", "1.2.3", true},
		{"latest", "1.2.3-beta.3", false},
		{"canary", "1.2.3-beta.3", false},
		{"1.2.3-draft", "1.2.3-draft", false},
	}
	for _, test := range tests {
		if got := immutableAlpha2Tag(test.tag, test.version); got != test.immutable {
			t.Errorf("immutableAlpha2Tag(%q, %q) = %v, want %v", test.tag, test.version, got, test.immutable)
		}
	}
}
