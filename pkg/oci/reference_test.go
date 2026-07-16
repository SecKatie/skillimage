package oci_test

import (
	"testing"

	"github.com/redhat-et/skillimage/pkg/oci"
)

func TestNormalizeBuildReferencePodmanDefaults(t *testing.T) {
	tests := map[string]string{
		"pdf-processing":              "localhost/pdf-processing:latest",
		"team/pdf-processing":         "localhost/team/pdf-processing:latest",
		"pdf-processing:canary":       "localhost/pdf-processing:canary",
		"ghcr.io/acme/pdf-processing": "ghcr.io/acme/pdf-processing:latest",
		"localhost:5000/pdf:dev":      "localhost:5000/pdf:dev",
	}
	for input, want := range tests {
		got, err := oci.NormalizeBuildReference(input)
		if err != nil {
			t.Fatalf("NormalizeBuildReference(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("NormalizeBuildReference(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeBuildReferenceRejectsDigest(t *testing.T) {
	if _, err := oci.NormalizeBuildReference("example/pdf@sha256:abc"); err == nil {
		t.Fatal("expected digest build target to fail")
	}
}

func TestIsManagedReference(t *testing.T) {
	tests := map[string]bool{
		"pdf-processing":                    true,
		"ghcr.io/acme/skills/pdf-processing": true,
		"localhost:5000/team/pdf-processing": true,
		"pdf-processing:canary":             false,
		"ghcr.io/acme/pdf:beta":             false,
		"ghcr.io/acme/pdf@sha256:abc":       false,
		"":                                  false,
	}
	for input, want := range tests {
		if got := oci.IsManagedReference(input); got != want {
			t.Errorf("IsManagedReference(%q) = %v, want %v", input, got, want)
		}
	}
}
