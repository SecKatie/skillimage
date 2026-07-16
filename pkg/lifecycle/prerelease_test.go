package lifecycle_test

import (
	"testing"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
)

func TestEffectiveVersion(t *testing.T) {
	tests := []struct {
		stage lifecycle.Stage
		num   int
		want  string
	}{
		{lifecycle.Alpha, 1, "1.2.3-alpha.1"},
		{lifecycle.Beta, 3, "1.2.3-beta.3"},
		{lifecycle.RC, 2, "1.2.3-rc.2"},
		{lifecycle.Final, 0, "1.2.3"},
	}
	for _, test := range tests {
		got, err := lifecycle.EffectiveVersion("1.2.3", test.stage, test.num)
		if err != nil {
			t.Fatalf("EffectiveVersion: %v", err)
		}
		if got != test.want {
			t.Errorf("EffectiveVersion(%s, %d) = %q, want %q", test.stage, test.num, got, test.want)
		}
	}
}

func TestEffectiveVersionRejectsNonCoreBaseAndZeroPrerelease(t *testing.T) {
	for _, base := range []string{"1.2", "v1.2.3", "1.2.3-beta.1", "1.2.3+build"} {
		if _, err := lifecycle.EffectiveVersion(base, lifecycle.Alpha, 1); err == nil {
			t.Errorf("expected %q to be rejected", base)
		}
	}
	if _, err := lifecycle.EffectiveVersion("1.2.3", lifecycle.Beta, 0); err == nil {
		t.Error("expected zero prerelease number to be rejected")
	}
}

func TestParseEffectiveVersion(t *testing.T) {
	base, stage, num, err := lifecycle.ParseEffectiveVersion("2.0.1-beta.12")
	if err != nil {
		t.Fatalf("ParseEffectiveVersion: %v", err)
	}
	if base != "2.0.1" || stage != lifecycle.Beta || num != 12 {
		t.Fatalf("got (%q, %q, %d), want (2.0.1, beta, 12)", base, stage, num)
	}
	if _, _, _, err := lifecycle.ParseEffectiveVersion("2.0.1-beta"); err == nil {
		t.Error("expected prerelease without numeric component to fail")
	}
}

func TestStageRanks(t *testing.T) {
	if lifecycle.StageRank(lifecycle.Alpha) >= lifecycle.StageRank(lifecycle.Beta) ||
		lifecycle.StageRank(lifecycle.Beta) >= lifecycle.StageRank(lifecycle.RC) ||
		lifecycle.StageRank(lifecycle.RC) >= lifecycle.StageRank(lifecycle.Final) {
		t.Fatal("stage rank must be alpha < beta < rc < final")
	}
}
