package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redhat-et/skillimage/pkg/oci"
)

func writeAlpha2Skill(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	card := `apiVersion: skillimage.io/v1alpha2
kind: SkillCard
metadata:
  version: 1.2.3
  title: PDF Processing
  vendor: Example Corp
  tags: [pdf]
  url: https://example.com/pdf
  documentation: https://docs.example.com/pdf
  support: https://example.com/support
  changelog: https://example.com/changelog
  annotations:
    com.example.reviewed: "true"
`
	skill := "---\nname: " + name + "\ndescription: Processes PDF documents.\nlicense: Apache-2.0\ncompatibility: Requires poppler.\nallowed-tools: Bash Read\nmetadata:\n  owner: docs-team\n---\n\n# Instructions\n\nProcess the PDF.\n"
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestBuildAlpha2DefaultCreatesVersionAndLatestThenIncrements(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	_, err = client.Build(context.Background(), dir, oci.BuildOptions{Tagged: func(ref string) { refs = append(refs, ref) }})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []string{"localhost/pdf-processing:1.2.3-alpha.1", "localhost/pdf-processing:latest"}
	if strings.Join(refs, ",") != strings.Join(want, ",") {
		t.Fatalf("refs = %v, want %v", refs, want)
	}
	result, err := client.Inspect(context.Background(), want[1])
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.2.3-alpha.1" || result.Status != "alpha" || result.AllowedTools != "Bash Read" || result.SkillCardVersion != "skillimage.io/v1alpha2" {
		t.Fatalf("unexpected inspect result: %#v", result)
	}
	if result.License != "Apache-2.0" {
		t.Fatalf("license annotation = %q", result.License)
	}
	if result.Vendor != "Example Corp" || result.URL != "https://example.com/pdf" || result.Documentation != "https://docs.example.com/pdf" || result.Support != "https://example.com/support" || result.Changelog != "https://example.com/changelog" {
		t.Fatalf("catalog annotations were not projected: %#v", result)
	}
	if result.Annotations["com.example.reviewed"] != "true" {
		t.Fatalf("custom annotation missing: %#v", result.Annotations)
	}

	refs = nil
	_, err = client.Build(context.Background(), dir, oci.BuildOptions{Tagged: func(ref string) { refs = append(refs, ref) }})
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if len(refs) != 2 || refs[0] != "localhost/pdf-processing:1.2.3-alpha.2" || refs[1] != "localhost/pdf-processing:latest" {
		t.Fatalf("second refs = %v", refs)
	}
}

func TestBuildAlpha2ExactTagCreatesOnlyExactReference(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	_, err = client.Build(context.Background(), dir, oci.BuildOptions{
		Tag:    "bumbleforge.com/kglitchy/skills/pdf-processing:canary",
		Tagged: func(ref string) { refs = append(refs, ref) },
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(refs) != 1 || refs[0] != "bumbleforge.com/kglitchy/skills/pdf-processing:canary" {
		t.Fatalf("refs = %v", refs)
	}
	images, err := client.ListLocal()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || images[0].Tag != "canary" {
		t.Fatalf("images = %#v", images)
	}
}

func TestBuildAlpha2ExactTagMayBeReplacedLocally(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ref := "bumbleforge.com/kglitchy/skills/pdf-processing:canary"
	first, err := client.Build(ctx, dir, oci.BuildOptions{Tag: ref})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "changed.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := client.Build(ctx, dir, oci.BuildOptions{Tag: ref})
	if err != nil {
		t.Fatalf("replacing exact local tag: %v", err)
	}
	if first.Digest == second.Digest {
		t.Fatal("replaced exact tag retained the old digest")
	}
}

func TestBuildAlpha2UntaggedTargetUsesManagedLifecycle(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	_, err = client.Build(context.Background(), dir, oci.BuildOptions{
		Tag:    "bumbleforge.com/kglitchy/skills/pdf-processing",
		Tagged: func(ref string) { refs = append(refs, ref) },
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []string{
		"bumbleforge.com/kglitchy/skills/pdf-processing:1.2.3-alpha.1",
		"bumbleforge.com/kglitchy/skills/pdf-processing:latest",
	}
	if strings.Join(refs, ",") != strings.Join(want, ",") {
		t.Fatalf("refs = %v, want %v", refs, want)
	}
}

func TestBuildAlpha2UntaggedTargetHonorsStageOverride(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	_, err = client.Build(context.Background(), dir, oci.BuildOptions{
		Tag:    "bumbleforge.com/kglitchy/skills/pdf-processing",
		Stage:  "beta",
		Tagged: func(ref string) { refs = append(refs, ref) },
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []string{
		"bumbleforge.com/kglitchy/skills/pdf-processing:1.2.3-beta.1",
		"bumbleforge.com/kglitchy/skills/pdf-processing:latest",
	}
	if strings.Join(refs, ",") != strings.Join(want, ",") {
		t.Fatalf("refs = %v, want %v", refs, want)
	}
}

func TestBuildAlpha2StageAndNumberOverrides(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var refs []string
	if _, err := client.Build(ctx, dir, oci.BuildOptions{Stage: "beta", Tagged: func(ref string) { refs = append(refs, ref) }}); err != nil {
		t.Fatal(err)
	}
	if refs[0] != "localhost/pdf-processing:1.2.3-beta.1" {
		t.Fatalf("stage override refs = %v", refs)
	}
	refs = nil
	if _, err := client.Build(ctx, dir, oci.BuildOptions{Stage: "rc", PrereleaseNumber: 3, Tagged: func(ref string) { refs = append(refs, ref) }}); err != nil {
		t.Fatal(err)
	}
	if refs[0] != "localhost/pdf-processing:1.2.3-rc.3" {
		t.Fatalf("number override refs = %v", refs)
	}
	if _, err := client.Build(ctx, dir, oci.BuildOptions{Stage: "rc", PrereleaseNumber: 3}); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("expected explicit-number collision, got %v", err)
	}
}

func TestBuildAlpha2IncludesDotfilesAndDereferencesSymlinks(t *testing.T) {
	parent := t.TempDir()
	dir := writeAlpha2Skill(t, parent, "pdf-processing")
	if err := os.WriteFile(filepath.Join(dir, ".skill-config"), []byte("hidden=true"), 0o644); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external.txt")
	if err := os.WriteFile(external, []byte("external contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(dir, "copied.txt")); err != nil {
		t.Fatal(err)
	}
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var warnings []string
	_, err = client.Build(context.Background(), dir, oci.BuildOptions{Warn: func(msg string) { warnings = append(warnings, msg) }})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(warnings) == 0 || !strings.Contains(strings.Join(warnings, "\n"), "external") {
		t.Fatalf("expected external symlink warning, got %v", warnings)
	}
	out := t.TempDir()
	if err := client.Unpack(context.Background(), "localhost/pdf-processing:latest", out); err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	root := filepath.Join(out, "pdf-processing")
	for path, want := range map[string]string{".skill-config": "hidden=true", "copied.txt": "external contents"} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || string(data) != want {
			t.Errorf("%s = %q, %v; want %q", path, data, err, want)
		}
		info, err := os.Lstat(filepath.Join(root, path))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s should be a regular copied file: %v, %v", path, info, err)
		}
	}
	info, err := os.Stat(filepath.Join(root, "copied.txt"))
	if err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("dereferenced executable mode was not preserved: %v, %v", info, err)
	}
}

func TestBuildAlpha2NonconformantRequiresFlag(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "description:", "unknown-field: value\ndescription:", 1))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Build(context.Background(), dir, oci.BuildOptions{}); err == nil {
		t.Fatal("expected strict build to reject unknown Agent Skills field")
	}
	if _, err := client.Build(context.Background(), dir, oci.BuildOptions{AllowNonconformant: true}); err != nil {
		t.Fatalf("allow-nonconformant Build: %v", err)
	}
}

func TestBuildAlpha2NonconformantNameFallsBackToDirectory(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "name: pdf-processing", "name: Wrong Name", 1))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	if _, err := client.Build(context.Background(), dir, oci.BuildOptions{AllowNonconformant: true, Tagged: func(ref string) { refs = append(refs, ref) }}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(refs) == 0 || refs[0] != "localhost/pdf-processing:1.2.3-alpha.1" {
		t.Fatalf("expected directory fallback reference, got %v", refs)
	}
}

func TestBuildAlpha2SupportsSymlinkedSkillRoot(t *testing.T) {
	target := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "pdf-processing")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Build(context.Background(), link, oci.BuildOptions{}); err != nil {
		t.Fatalf("Build symlinked root: %v", err)
	}
}

func TestBuildAlpha2RejectsBrokenAndCyclicSymlinks(t *testing.T) {
	for _, test := range []struct {
		name   string
		target string
	}{
		{"broken", "missing-target"},
		{"cycle", "."},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
			if err := os.Symlink(test.target, filepath.Join(dir, "bad-link")); err != nil {
				t.Fatal(err)
			}
			client, err := oci.NewClient(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Build(context.Background(), dir, oci.BuildOptions{}); err == nil {
				t.Fatalf("expected %s symlink to fail", test.name)
			}
		})
	}
}
