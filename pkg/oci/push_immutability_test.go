package oci

import "testing"

func TestImmutableAlpha2Tag(t *testing.T) {
	tests := []struct {
		tag       string
		immutable bool
	}{
		{"1.2.3-alpha.1", true},
		{"1.2.3-beta.3", true},
		{"1.2.3", true},
		{"latest", false},
		{"canary", false},
		{"1.2.3-draft", false},
	}
	for _, test := range tests {
		if got := immutableAlpha2Tag(test.tag); got != test.immutable {
			t.Errorf("immutableAlpha2Tag(%q) = %v, want %v", test.tag, got, test.immutable)
		}
	}
}
