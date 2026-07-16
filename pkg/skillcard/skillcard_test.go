package skillcard_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/redhat-et/skillimage/pkg/skillcard"
)

const validSkillYAML = `apiVersion: skillimage.io/v1alpha1
kind: SkillCard
metadata:
  name: hello-world
  namespace: examples
  version: 1.0.0
  display-name: "Hello World"
  description: A simple example skill.
  license: Apache-2.0
  tags:
    - example
  authors:
    - name: Test Author
      email: test@example.com
spec:
  prompt: SKILL.md
`

const validV1Alpha2YAML = `apiVersion: skillimage.io/v1alpha2
kind: SkillCard
metadata:
  version: 1.2.3
  title: PDF Processing
  vendor: Example Corp
  tags: [pdf, documents]
  authors:
    - name: Test Author
      email: test@example.com
  url: https://example.com/pdf
  documentation: https://docs.example.com/pdf
  support: https://example.com/support
  changelog: https://example.com/changelog
  annotations:
    com.example.security.reviewed: "true"
`

func TestParse(t *testing.T) {
	sc, err := skillcard.Parse(strings.NewReader(validSkillYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.APIVersion != "skillimage.io/v1alpha1" {
		t.Errorf("apiVersion = %q, want %q", sc.APIVersion, "skillimage.io/v1alpha1")
	}
	if sc.Kind != "SkillCard" {
		t.Errorf("kind = %q, want %q", sc.Kind, "SkillCard")
	}
	if sc.Metadata.Name != "hello-world" {
		t.Errorf("name = %q, want %q", sc.Metadata.Name, "hello-world")
	}
	if sc.Metadata.Namespace != "examples" {
		t.Errorf("namespace = %q, want %q", sc.Metadata.Namespace, "examples")
	}
	if sc.Metadata.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", sc.Metadata.Version, "1.0.0")
	}
	if sc.Metadata.DisplayName != "Hello World" {
		t.Errorf("display-name = %q, want %q", sc.Metadata.DisplayName, "Hello World")
	}
	if len(sc.Metadata.Authors) != 1 || sc.Metadata.Authors[0].Name != "Test Author" {
		t.Errorf("authors = %v, want [{Test Author test@example.com}]", sc.Metadata.Authors)
	}
	if sc.Spec == nil || sc.Spec.Prompt != "SKILL.md" {
		t.Errorf("spec.prompt = %v, want SKILL.md", sc.Spec)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	_, err := skillcard.Parse(strings.NewReader("not: [valid: yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSerialize(t *testing.T) {
	sc, err := skillcard.Parse(strings.NewReader(validSkillYAML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := skillcard.Serialize(sc, &buf); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	roundtrip, err := skillcard.Parse(&buf)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if roundtrip.Metadata.Name != sc.Metadata.Name {
		t.Errorf("roundtrip name = %q, want %q", roundtrip.Metadata.Name, sc.Metadata.Name)
	}
	if roundtrip.Metadata.Version != sc.Metadata.Version {
		t.Errorf("roundtrip version = %q, want %q", roundtrip.Metadata.Version, sc.Metadata.Version)
	}
}

func TestValidateValid(t *testing.T) {
	sc, err := skillcard.Parse(strings.NewReader(validSkillYAML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	errs, err := skillcard.Validate(sc)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got %v", errs)
	}
}

func TestValidateMissingRequiredFields(t *testing.T) {
	yaml := `apiVersion: skillimage.io/v1alpha1
kind: SkillCard
metadata:
  name: test
`
	sc, err := skillcard.Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	errs, err := skillcard.Validate(sc)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected validation errors for missing required fields")
	}
	for _, required := range []string{"namespace", "version", "description"} {
		found := false
		for _, e := range errs {
			if strings.Contains(e.Field, required) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error for missing %q, got errors: %v", required, errs)
		}
	}
}

func TestValidateInvalidName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "hello-world", false},
		{"valid single char", "a", false},
		{"uppercase", "Hello", true},
		{"spaces", "hello world", true},
		{"leading hyphen", "-hello", true},
		{"trailing hyphen", "hello-", true},
		{"consecutive hyphens", "hello--world", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := fmt.Sprintf(`apiVersion: skillimage.io/v1alpha1
kind: SkillCard
metadata:
  name: %s
  namespace: test
  version: 1.0.0
  description: test
`, tt.value)
			sc, err := skillcard.Parse(strings.NewReader(yaml))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			errs, _ := skillcard.Validate(sc)
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("name=%q: hasErr=%v, wantErr=%v, errs=%v",
					tt.value, hasErr, tt.wantErr, errs)
			}
		})
	}
}

func TestValidateInvalidSemver(t *testing.T) {
	yaml := `apiVersion: skillimage.io/v1alpha1
kind: SkillCard
metadata:
  name: test
  namespace: test
  version: not-semver
  description: test
`
	sc, err := skillcard.Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	errs, _ := skillcard.Validate(sc)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Field, "version") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected semver validation error, got: %v", errs)
	}
}

func TestValidateWrongAPIVersion(t *testing.T) {
	yaml := `apiVersion: wrong/v1
kind: SkillCard
metadata:
  name: test
  namespace: test
  version: 1.0.0
  description: test
`
	sc, err := skillcard.Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	errs, _ := skillcard.Validate(sc)
	if len(errs) == 0 {
		t.Fatal("expected error for wrong apiVersion")
	}
}

func TestUnknownVersionWithAlpha2FieldsReturnsSupportedVersionFinding(t *testing.T) {
	sc, err := skillcard.Parse(strings.NewReader(`apiVersion: skillimage.io/v9
kind: SkillCard
metadata:
  version: 1.0.0
  title: Future Card
`))
	if err != nil {
		t.Fatalf("Parse should preserve unknown version for validation: %v", err)
	}
	errs, err := skillcard.Validate(sc)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Message, skillcard.APIVersionV1Alpha1) || !strings.Contains(errs[0].Message, skillcard.APIVersionV1Alpha2) {
		t.Fatalf("unexpected validation findings: %#v", errs)
	}
}

func TestParseSerializeV1Alpha2(t *testing.T) {
	sc, err := skillcard.Parse(strings.NewReader(validV1Alpha2YAML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if sc.APIVersion != skillcard.APIVersionV1Alpha2 || sc.Metadata.Version != "1.2.3" {
		t.Fatalf("unexpected parsed card: %#v", sc)
	}
	if sc.Metadata.Name != "" || sc.Metadata.Namespace != "" || sc.Spec != nil || sc.Provenance != nil {
		t.Fatalf("alpha2 populated removed fields: %#v", sc)
	}
	if sc.Metadata.Annotations["com.example.security.reviewed"] != "true" {
		t.Fatalf("annotation was not preserved: %#v", sc.Metadata.Annotations)
	}
	var out bytes.Buffer
	if err := skillcard.Serialize(sc, &out); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	for _, removed := range []string{"namespace:", "description:", "license:", "allowed-tools:", "spec:", "provenance:"} {
		if strings.Contains(out.String(), removed) {
			t.Errorf("serialized alpha2 contains removed field %q:\n%s", removed, out.String())
		}
	}
}

func TestV1Alpha2RequiresCoreSemver(t *testing.T) {
	for _, version := range []string{"1.2", "v1.2.3", "1.2.3-beta.1", "1.2.3+build"} {
		yaml := strings.Replace(validV1Alpha2YAML, "version: 1.2.3", `version: "`+version+`"`, 1)
		sc, err := skillcard.Parse(strings.NewReader(yaml))
		if err != nil {
			t.Fatalf("Parse(%q): %v", version, err)
		}
		errs, err := skillcard.Validate(sc)
		if err != nil {
			t.Fatalf("Validate(%q): %v", version, err)
		}
		if len(errs) == 0 {
			t.Errorf("expected version %q to be rejected", version)
		}
	}
}

func TestV1Alpha2RejectsRemovedAndInvalidAnnotationFields(t *testing.T) {
	withRemoved := strings.Replace(validV1Alpha2YAML, "  version: 1.2.3", "  version: 1.2.3\n  namespace: legacy", 1)
	if _, err := skillcard.Parse(strings.NewReader(withRemoved)); err == nil {
		t.Fatal("expected removed alpha2 field to fail strict parsing")
	}

	badKey := strings.Replace(validV1Alpha2YAML, "com.example.security.reviewed", "not-reverse-domain", 1)
	sc, err := skillcard.Parse(strings.NewReader(badKey))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	errs, err := skillcard.Validate(sc)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected invalid annotation key to fail")
	}
}

func TestV1Alpha2RejectsNonStringCatalogValuesWithoutYAMLCoercion(t *testing.T) {
	for name, yaml := range map[string]string{
		"annotation": strings.Replace(validV1Alpha2YAML, `com.example.security.reviewed: "true"`, `com.example.security.reviewed: true`, 1),
		"title":      strings.Replace(validV1Alpha2YAML, "title: PDF Processing", "title: 123", 1),
		"tag":        strings.Replace(validV1Alpha2YAML, "tags: [pdf, documents]", "tags: [pdf, 123]", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := skillcard.Parse(strings.NewReader(yaml)); err == nil {
				t.Fatal("expected non-string catalog value to fail parsing")
			}
		})
	}
}
