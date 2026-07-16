package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/redhat-et/skillimage/pkg/oci"
)

func TestBuildAlpha2Flags(t *testing.T) {
	cmd := newBuildCmd()
	for _, name := range []string{"tag", "stage", "prerelease-number", "allow-nonconformant", "push", "force", "tls-verify"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("build is missing --%s", name)
		}
	}
	if got := cmd.Flags().ShorthandLookup("t"); got == nil || got.Name != "tag" {
		t.Fatal("build -t must be shorthand for --tag")
	}
}

func TestManagedSyncFlags(t *testing.T) {
	for command, cmd := range map[string]*cobra.Command{
		"push":    newPushCmd(),
		"pull":    newPullCmd(),
		"promote": newPromoteCmd(),
	} {
		if cmd.Flags().Lookup("force") == nil {
			t.Errorf("%s is missing --force", command)
		}
	}
	if newPromoteCmd().Flags().Lookup("push") == nil {
		t.Error("promote is missing --push")
	}
}

func TestForceRequiresPushOnBuildAndPromote(t *testing.T) {
	for name, cmd := range map[string]*cobra.Command{
		"build":   newBuildCmd(),
		"promote": newPromoteCmd(),
	} {
		cmd.SetArgs([]string{"--force", "unused"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "--force requires --push") {
			t.Errorf("%s --force error = %v", name, err)
		}
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

func TestPromoteHelpDescribesAlpha2Lifecycle(t *testing.T) {
	help := newPromoteCmd().Long
	for _, want := range []string{
		"alpha -> beta -> rc -> final",
		"automatically advances one stage",
		"skillctl promote bumbleforge.com/kglitchy/skills/feature-brainstorming",
		"unless --push is used",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("promote help does not contain %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "State transitions: draft -> testing") {
		t.Fatalf("promote help leads with the legacy v1alpha1 lifecycle:\n%s", help)
	}
}

func TestDemoteHelpDescribesAlpha2Lifecycle(t *testing.T) {
	help := newDemoteCmd().Long
	for _, want := range []string{
		"alpha < beta < rc < final",
		"operates on local latest",
		"skillctl demote bumbleforge.com/kglitchy/skills/feature-brainstorming --to beta",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("demote help does not contain %q:\n%s", want, help)
		}
	}
}

func TestPromoteDefaultsToLocalNextStage(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := filepath.Join(t.TempDir(), "managed-skill")
	if err := runInit(&cobra.Command{}, dir, initOptions{name: "managed-skill", version: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	client, err := defaultClient()
	if err != nil {
		t.Fatal(err)
	}
	repository := "registry.example/team/managed-skill"
	if _, err := client.Build(context.Background(), dir, oci.BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}

	cmd := newPromoteCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{repository})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("promote: %v\n%s", err, stdout.String())
	}
	client, err = defaultClient()
	if err != nil {
		t.Fatal(err)
	}
	latest, err := client.Inspect(context.Background(), repository+":latest")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != "0.1.0-beta.1" {
		t.Fatalf("latest = %s, output:\n%s", latest.Version, stdout.String())
	}
}
