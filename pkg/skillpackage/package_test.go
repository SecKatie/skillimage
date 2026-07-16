package skillpackage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/redhat-et/skillimage/pkg/skillpackage"
)

func TestLoadCurrentDirectoryUsesActualBasename(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "feature-brainstorming")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte("apiVersion: skillimage.io/v1alpha2\nkind: SkillCard\nmetadata:\n  version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: feature-brainstorming\ndescription: Designs features.\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	pkg, err := skillpackage.Load(".", skillpackage.Options{})
	if err != nil {
		t.Fatalf("Load from current directory: %v", err)
	}
	if pkg.Name() != "feature-brainstorming" {
		t.Fatalf("Name = %q", pkg.Name())
	}
}
