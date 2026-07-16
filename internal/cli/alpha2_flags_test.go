package cli

import (
	"strings"
	"testing"
)

func TestBuildAlpha2Flags(t *testing.T) {
	cmd := newBuildCmd()
	for _, name := range []string{"tag", "stage", "prerelease-number", "allow-nonconformant"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("build is missing --%s", name)
		}
	}
	if got := cmd.Flags().ShorthandLookup("t"); got == nil || got.Name != "tag" {
		t.Fatal("build -t must be shorthand for --tag")
	}
}

func TestBuildExplicitPrereleaseNumberMustBePositive(t *testing.T) {
	root := NewRootCmd("test")
	root.SetArgs([]string{"build", "--prerelease-number", "0", "unused"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected explicit zero to fail, got %v", err)
	}
}

func TestValidateAllowNonconformantFlag(t *testing.T) {
	if newValidateCmd().Flags().Lookup("allow-nonconformant") == nil {
		t.Fatal("validate is missing --allow-nonconformant")
	}
}

func TestDemoteCommandIsRegistered(t *testing.T) {
	root := NewRootCmd("test")
	if cmd, _, err := root.Find([]string{"demote"}); err != nil || cmd == root {
		t.Fatalf("demote command is not registered: %v", err)
	}
}
