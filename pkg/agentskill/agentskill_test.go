package agentskill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redhat-et/skillimage/pkg/agentskill"
)

func TestParseAllAgentSkillsFields(t *testing.T) {
	input := []byte("---\r\nname: pdf-processing\r\ndescription: Processes PDF files.\r\nlicense: Apache-2.0\r\ncompatibility: Requires poppler.\r\nmetadata:\r\n  owner: docs-team\r\nallowed-tools: Bash Read\r\n---\r\n\r\n# Instructions\r\n\r\nDo the work.\r\n")
	skill, err := agentskill.Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if skill.Name != "pdf-processing" || skill.License != "Apache-2.0" || skill.Metadata["owner"] != "docs-team" || skill.AllowedTools != "Bash Read" {
		t.Fatalf("unexpected skill: %#v", skill)
	}
	if skill.Body != "# Instructions\n\nDo the work." {
		t.Fatalf("body = %q", skill.Body)
	}
}

func TestValidateStrictAndDirectoryName(t *testing.T) {
	skill, err := agentskill.Parse([]byte("---\nname: pdf-processing\ndescription: Processes PDFs.\nextra-field: no\n---\nBody\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	findings := agentskill.Validate(skill, filepath.Join("tmp", "different-name"))
	joined := ""
	for _, finding := range findings {
		joined += finding.String() + "\n"
	}
	for _, want := range []string{"parent directory", "extra-field"} {
		if !strings.Contains(joined, want) {
			t.Errorf("findings %q do not contain %q", joined, want)
		}
	}
}

func TestParseFileRequiresSkillMD(t *testing.T) {
	_, err := agentskill.ParseFile(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "SKILL.md") {
		t.Fatalf("expected missing SKILL.md error, got %v", err)
	}
}

func TestValidateMetadataValuesMustBeStrings(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pdf-processing")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := agentskill.Parse([]byte("---\nname: pdf-processing\ndescription: Processes PDFs.\nmetadata:\n  count: 3\n---\nBody\n"))
	if err == nil {
		t.Fatal("expected non-string metadata value to fail")
	}
}
