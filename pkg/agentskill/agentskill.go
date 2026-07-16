package agentskill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var knownFields = map[string]struct{}{
	"name": {}, "description": {}, "license": {}, "compatibility": {},
	"metadata": {}, "allowed-tools": {},
}

type Skill struct {
	Name          string            `yaml:"name" json:"name"`
	Description   string            `yaml:"description" json:"description"`
	License       string            `yaml:"license,omitempty" json:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`
	Body          string            `yaml:"-" json:"body"`
	UnknownFields []string          `yaml:"-" json:"unknown_fields,omitempty"`
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func ParseFile(skillDir string) (*Skill, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Skill, error) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := yaml.Unmarshal(frontmatter, &raw); err != nil {
		return nil, fmt.Errorf("parsing SKILL.md frontmatter: %w", err)
	}
	if metadata, ok := raw["metadata"]; ok {
		entries, ok := metadata.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("parsing SKILL.md frontmatter: metadata must be a string-to-string mapping")
		}
		for key, value := range entries {
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("parsing SKILL.md frontmatter: metadata.%s must be a string", key)
			}
		}
	}

	var skill Skill
	dec := yaml.NewDecoder(bytes.NewReader(frontmatter))
	if err := dec.Decode(&skill); err != nil {
		return nil, fmt.Errorf("parsing SKILL.md frontmatter: %w", err)
	}
	skill.Body = strings.TrimSpace(string(body))
	for key := range raw {
		if _, ok := knownFields[key]; !ok {
			skill.UnknownFields = append(skill.UnknownFields, key)
		}
	}
	sort.Strings(skill.UnknownFields)
	return &skill, nil
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(normalized, []byte("---\n")) {
		return nil, nil, fmt.Errorf("SKILL.md must start with YAML frontmatter delimited by ---")
	}
	rest := normalized[4:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx < 0 {
		return nil, nil, fmt.Errorf("SKILL.md frontmatter is missing its closing --- delimiter")
	}
	end := idx + 4
	if len(rest) > end && rest[end] != '\n' {
		return nil, nil, fmt.Errorf("SKILL.md closing --- delimiter must be on its own line")
	}
	bodyStart := end
	if len(rest) > bodyStart && rest[bodyStart] == '\n' {
		bodyStart++
	}
	return rest[:idx], rest[bodyStart:], nil
}

func Validate(skill *Skill, skillDir string) []ValidationError {
	var errs []ValidationError
	dirName := DirectoryName(skillDir)
	if n := utf8.RuneCountInString(skill.Name); n < 1 || n > 64 {
		errs = append(errs, ValidationError{"name", "must be between 1 and 64 characters"})
	} else if !validName.MatchString(skill.Name) {
		errs = append(errs, ValidationError{"name", "must contain only lowercase letters, numbers, and single hyphens"})
	}
	if skill.Name != "" && dirName != skill.Name {
		errs = append(errs, ValidationError{"name", fmt.Sprintf("must match parent directory name %q", dirName)})
	}
	if n := utf8.RuneCountInString(skill.Description); n < 1 || n > 1024 {
		errs = append(errs, ValidationError{"description", "must be between 1 and 1024 characters"})
	}
	if skill.Compatibility != "" {
		if n := utf8.RuneCountInString(skill.Compatibility); n > 500 {
			errs = append(errs, ValidationError{"compatibility", "must not exceed 500 characters"})
		}
	}
	for _, field := range skill.UnknownFields {
		errs = append(errs, ValidationError{field, "unknown Agent Skills frontmatter field"})
	}
	return errs
}

// HasUsableName reports whether name is conformant and identifies this logical
// skill directory. Nonconformant package modes use the directory name when it
// returns false.
func HasUsableName(skill *Skill, skillDir string) bool {
	n := utf8.RuneCountInString(skill.Name)
	return n >= 1 && n <= 64 && validName.MatchString(skill.Name) && skill.Name == DirectoryName(skillDir)
}

// DirectoryName returns the logical basename for a skill path. Resolving to an
// absolute path turns "." into the current directory name without evaluating
// intentional symlinks supplied by the user.
func DirectoryName(skillDir string) string {
	abs, err := filepath.Abs(skillDir)
	if err == nil {
		return filepath.Base(filepath.Clean(abs))
	}
	return filepath.Base(filepath.Clean(skillDir))
}
