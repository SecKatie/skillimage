package v1alpha2

type SkillCard struct {
	APIVersion string   `yaml:"apiVersion" json:"api_version"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
}

type Metadata struct {
	Version       string            `yaml:"version" json:"version"`
	Title         string            `yaml:"title,omitempty" json:"title,omitempty"`
	Vendor        string            `yaml:"vendor,omitempty" json:"vendor,omitempty"`
	Tags          []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Authors       []Author          `yaml:"authors,omitempty" json:"authors,omitempty"`
	URL           string            `yaml:"url,omitempty" json:"url,omitempty"`
	Documentation string            `yaml:"documentation,omitempty" json:"documentation,omitempty"`
	Support       string            `yaml:"support,omitempty" json:"support,omitempty"`
	Changelog     string            `yaml:"changelog,omitempty" json:"changelog,omitempty"`
	Annotations   map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type Author struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
}
