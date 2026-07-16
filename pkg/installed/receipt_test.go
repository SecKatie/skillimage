package installed_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/redhat-et/skillimage/pkg/installed"
)

func TestWriteReceiptOutsideSkillContentAndScanAlpha2(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "pdf-processing")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	card := "apiVersion: skillimage.io/v1alpha2\nkind: SkillCard\nmetadata:\n  version: 1.2.3\n"
	skill := "---\nname: pdf-processing\ndescription: Processes PDF files.\n---\nInstructions\n"
	if err := os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	want := installed.Receipt{
		Name:        "pdf-processing",
		Version:     "1.2.3-beta.2",
		Source:      "ghcr.io/acme/pdf-processing:latest",
		Digest:      "sha256:abc",
		InstalledAt: time.Unix(123, 0).UTC(),
	}
	if err := installed.WriteReceipt(root, want); err != nil {
		t.Fatalf("WriteReceipt: %v", err)
	}
	receiptPath := filepath.Join(root, ".skillimage", "receipts", "pdf-processing.json")
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var got installed.Receipt
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != want.Version || got.Source != want.Source || got.Digest != want.Digest {
		t.Fatalf("receipt = %#v, want %#v", got, want)
	}
	cardAfter, err := os.ReadFile(filepath.Join(skillDir, "skill.yaml"))
	if err != nil || string(cardAfter) != card {
		t.Fatalf("skill content was mutated: %q, %v", cardAfter, err)
	}

	skills, err := installed.Scan(map[string]string{"custom": root})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills = %#v", skills)
	}
	if skills[0].Name != want.Name || skills[0].Version != want.Version || skills[0].Source != want.Source || skills[0].Digest != want.Digest {
		t.Fatalf("scanned skill = %#v", skills[0])
	}
}

func TestWriteReceiptRejectsUnsafeName(t *testing.T) {
	if err := installed.WriteReceipt(t.TempDir(), installed.Receipt{Name: "../escape"}); err == nil {
		t.Fatal("expected unsafe receipt name to fail")
	}
}
