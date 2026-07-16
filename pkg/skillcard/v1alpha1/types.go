package v1alpha1

type SkillCard struct {
	APIVersion string      `yaml:"apiVersion" json:"api_version"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Provenance *Provenance `yaml:"provenance,omitempty" json:"provenance,omitempty"`
	Spec       *Spec       `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type Metadata struct {
	Name          string   `yaml:"name" json:"name"`
	DisplayName   string   `yaml:"display-name,omitempty" json:"display_name,omitempty"`
	Namespace     string   `yaml:"namespace" json:"namespace"`
	Version       string   `yaml:"version" json:"version"`
	Description   string   `yaml:"description" json:"description"`
	License       string   `yaml:"license,omitempty" json:"license,omitempty"`
	Compatibility string   `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Tags          []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Authors       []Author `yaml:"authors,omitempty" json:"authors,omitempty"`
	AllowedTools  string   `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`
}

type Author struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
}

type Provenance struct {
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
	Commit string `yaml:"commit,omitempty" json:"commit,omitempty"`
	Path   string `yaml:"path,omitempty" json:"path,omitempty"`
}

type Spec struct {
	Prompt       string       `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	Examples     []Example    `yaml:"examples,omitempty" json:"examples,omitempty"`
	Dependencies []Dependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

type Example struct {
	Input  string `yaml:"input" json:"input"`
	Output string `yaml:"output" json:"output"`
}

type Dependency struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}
