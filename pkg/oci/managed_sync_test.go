package oci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
)

func writeManagedTestSkill(t *testing.T, parent, body string) string {
	t.Helper()
	dir := filepath.Join(parent, "managed-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	card := `apiVersion: skillimage.io/v1alpha2
kind: SkillCard
metadata:
  version: 0.1.0
`
	skill := "---\nname: managed-skill\ndescription: Exercises managed OCI synchronization.\n---\n\n# Instructions\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func newManagedTestClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestPushManagedPublishesEffectiveVersionAndLatest(t *testing.T) {
	ctx := context.Background()
	local := newManagedTestClient(t)
	remote := newManagedTestClient(t)
	dir := writeManagedTestSkill(t, t.TempDir(), "alpha one")
	repository := "registry.example/team/managed-skill"
	if _, err := local.Build(ctx, dir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}

	result, err := pushManaged(ctx, local.store, remote.store, repository, false)
	if err != nil {
		t.Fatalf("pushManaged: %v", err)
	}
	if result.Version != "0.1.0-alpha.1" {
		t.Fatalf("version = %q", result.Version)
	}
	for _, ref := range []string{repository + ":0.1.0-alpha.1", repository + ":latest"} {
		if _, err := inspect(ctx, remote.store, ref); err != nil {
			t.Fatalf("remote %s: %v", ref, err)
		}
	}
}

func TestPushManagedAllowsForwardProgressAndRefusesRemoteAhead(t *testing.T) {
	ctx := context.Background()
	local := newManagedTestClient(t)
	remote := newManagedTestClient(t)
	dir := writeManagedTestSkill(t, t.TempDir(), "alpha builds")
	repository := "registry.example/team/managed-skill"

	if _, err := local.Build(ctx, dir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}
	if _, err := pushManaged(ctx, local.store, remote.store, repository, false); err != nil {
		t.Fatal(err)
	}
	if _, err := local.Build(ctx, dir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}
	if _, err := pushManaged(ctx, local.store, remote.store, repository, false); err != nil {
		t.Fatalf("forward push: %v", err)
	}

	if _, err := remote.Build(ctx, dir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}
	_, err := pushManaged(ctx, local.store, remote.store, repository, false)
	if err == nil || !strings.Contains(err.Error(), "skillctl pull "+repository) {
		t.Fatalf("expected remote-ahead recovery error, got %v", err)
	}
}

func TestPushManagedDigestConflictRequiresForce(t *testing.T) {
	ctx := context.Background()
	local := newManagedTestClient(t)
	remote := newManagedTestClient(t)
	repository := "registry.example/team/managed-skill"
	localDir := writeManagedTestSkill(t, t.TempDir(), "local contents")
	remoteDir := writeManagedTestSkill(t, t.TempDir(), "remote contents")
	if _, err := local.Build(ctx, localDir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Build(ctx, remoteDir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}

	_, err := pushManaged(ctx, local.store, remote.store, repository, false)
	if err == nil || !strings.Contains(err.Error(), "different content") || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected digest conflict, got %v", err)
	}
	result, err := pushManaged(ctx, local.store, remote.store, repository, true)
	if err != nil {
		t.Fatalf("forced push: %v", err)
	}
	if len(result.Replacements) == 0 {
		t.Fatal("forced push did not report replacements")
	}
	localLatest, _ := inspect(ctx, local.store, repository+":latest")
	remoteLatest, _ := inspect(ctx, remote.store, repository+":latest")
	if localLatest.Digest != remoteLatest.Digest {
		t.Fatalf("forced push did not replace remote latest: local %s remote %s", localLatest.Digest, remoteLatest.Digest)
	}
}

func TestPushManagedRefusesInconsistentLocalVersionTag(t *testing.T) {
	ctx := context.Background()
	local := newManagedTestClient(t)
	remote := newManagedTestClient(t)
	repository := "registry.example/team/managed-skill"
	dir := writeManagedTestSkill(t, t.TempDir(), "managed contents")
	if _, err := local.Build(ctx, dir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "changed.txt"), []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := local.Build(ctx, dir, BuildOptions{Tag: repository + ":0.1.0-alpha.1"}); err != nil {
		t.Fatal(err)
	}
	_, err := pushManaged(ctx, local.store, remote.store, repository, false)
	if err == nil || !strings.Contains(err.Error(), "local latest") || !strings.Contains(err.Error(), "version tag") {
		t.Fatalf("expected inconsistent local cursor error, got %v", err)
	}
}

func TestPullManagedRestoresVersionAndLatestAndProtectsLocalProgress(t *testing.T) {
	ctx := context.Background()
	local := newManagedTestClient(t)
	remote := newManagedTestClient(t)
	repository := "registry.example/team/managed-skill"
	dir := writeManagedTestSkill(t, t.TempDir(), "remote beta")
	if _, err := remote.Build(ctx, dir, BuildOptions{Tag: repository, Stage: "beta"}); err != nil {
		t.Fatal(err)
	}

	result, err := pullManaged(ctx, remote.store, local.store, repository, false)
	if err != nil {
		t.Fatalf("pullManaged: %v", err)
	}
	if result.Version != "0.1.0-beta.1" {
		t.Fatalf("version = %q", result.Version)
	}
	for _, ref := range []string{repository + ":0.1.0-beta.1", repository + ":latest"} {
		if _, err := inspect(ctx, local.store, ref); err != nil {
			t.Fatalf("local %s: %v", ref, err)
		}
	}

	if err := local.PromoteStageLocal(ctx, repository+":latest", lifecycle.RC); err != nil {
		t.Fatal(err)
	}
	_, err = pullManaged(ctx, remote.store, local.store, repository, false)
	if err == nil || !strings.Contains(err.Error(), "local latest") || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected local-ahead refusal, got %v", err)
	}
	if _, err := pullManaged(ctx, remote.store, local.store, repository, true); err != nil {
		t.Fatalf("forced pull: %v", err)
	}
	got, _ := inspect(ctx, local.store, repository+":latest")
	if got.Version != "0.1.0-beta.1" {
		t.Fatalf("forced pull latest = %s", got.Version)
	}
}

func TestPromoteManagedAdvancesAutomatically(t *testing.T) {
	ctx := context.Background()
	client := newManagedTestClient(t)
	dir := writeManagedTestSkill(t, t.TempDir(), "promotion")
	repository := "registry.example/team/managed-skill"
	if _, err := client.Build(ctx, dir, BuildOptions{Tag: repository}); err != nil {
		t.Fatal(err)
	}

	wants := []string{"0.1.0-beta.1", "0.1.0-rc.1", "0.1.0"}
	for _, want := range wants {
		result, err := client.PromoteManagedLocal(ctx, repository, "")
		if err != nil {
			t.Fatalf("promote to %s: %v", want, err)
		}
		if result.Version != want {
			t.Fatalf("promoted version = %s, want %s", result.Version, want)
		}
	}
	if _, err := client.PromoteManagedLocal(ctx, repository, ""); err == nil || !strings.Contains(err.Error(), "base version") {
		t.Fatalf("expected final promotion failure, got %v", err)
	}
}

func TestDemoteManagedUsesLocalLatest(t *testing.T) {
	ctx := context.Background()
	client := newManagedTestClient(t)
	dir := writeManagedTestSkill(t, t.TempDir(), "demotion")
	repository := "registry.example/team/managed-skill"
	if _, err := client.Build(ctx, dir, BuildOptions{Tag: repository, Stage: "rc"}); err != nil {
		t.Fatal(err)
	}
	result, err := client.DemoteManagedLocal(ctx, repository, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "0.1.0-beta.1" {
		t.Fatalf("demoted version = %s", result.Version)
	}
}

func TestRemoteVersionTagProtectionDoesNotDependOnManifestCursor(t *testing.T) {
	if !immutableAlpha2Tag("0.1.0-beta.2") {
		t.Fatal("a semantic version tag must be protected even when an exact local build has different lifecycle metadata")
	}
	if immutableAlpha2Tag("canary") {
		t.Fatal("custom aliases must remain mutable")
	}
}
