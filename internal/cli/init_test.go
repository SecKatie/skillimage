package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redhat-et/skillimage/pkg/skillcard"
	"github.com/spf13/cobra"
)

func TestInitCreatesAlpha2Pair(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pdf-processing")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := runInit(cmd, dir, initOptions{version: "0.1.0"}); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	cardFile, err := os.Open(filepath.Join(dir, "skill.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cardFile.Close() }()
	card, err := skillcard.Parse(cardFile)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if card.APIVersion != skillcard.APIVersionV1Alpha2 || card.Metadata.Version != "0.1.0" {
		t.Fatalf("unexpected card: %#v", card)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestInitFullCreatesFunctionalExamples(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pdf-processing")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := runInit(cmd, dir, initOptions{version: "1.0.0", full: true}); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	for _, path := range []string{"scripts/example.sh", "references/REFERENCE.md", "assets/example-template.md"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("expected %s: %v", path, err)
		}
	}
	info, err := os.Stat(filepath.Join(dir, "scripts", "example.sh"))
	if err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("example script is not executable: %v, %v", info, err)
	}
	output, err := exec.Command(filepath.Join(dir, "scripts", "example.sh"), "test-value").CombinedOutput()
	if err != nil || string(output) != "{\"ok\":true,\"value\":\"test-value\"}\n" {
		t.Fatalf("example script output = %q, %v", output, err)
	}
}

func TestInitCollisionListsFilesAndForceReplacesManagedFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pdf-processing")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cardPath := filepath.Join(dir, "skill.yaml")
	if err := os.WriteFile(cardPath, []byte("do not overwrite"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := runInit(cmd, dir, initOptions{version: "0.1.0"})
	if err == nil || !strings.Contains(err.Error(), cardPath) || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected collision details, got %v", err)
	}
	if err := runInit(cmd, dir, initOptions{version: "0.1.0", force: true}); err != nil {
		t.Fatalf("forced init: %v", err)
	}
	data, err := os.ReadFile(cardPath)
	if err != nil || !strings.Contains(string(data), skillcard.APIVersionV1Alpha2) {
		t.Fatalf("forced init did not replace card: %q, %v", data, err)
	}
}

func TestInitCurrentDirectoryUsesActualBasename(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "feature-brainstorming")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := runInit(cmd, ".", initOptions{version: "0.1.0"}); err != nil {
		t.Fatalf("runInit from current directory: %v", err)
	}
	data, err := os.ReadFile("SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: feature-brainstorming") {
		t.Fatalf("generated SKILL.md used the wrong name:\n%s", data)
	}
}
