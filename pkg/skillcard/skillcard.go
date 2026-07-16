package skillcard

import (
	"bytes"
	"fmt"
	"io"

	v1alpha1 "github.com/redhat-et/skillimage/pkg/skillcard/v1alpha1"
	v1alpha2 "github.com/redhat-et/skillimage/pkg/skillcard/v1alpha2"
	"gopkg.in/yaml.v3"
)

const (
	APIVersionV1Alpha1 = "skillimage.io/v1alpha1"
	APIVersionV1Alpha2 = "skillimage.io/v1alpha2"
)

type SkillCard struct {
	APIVersion string      `yaml:"apiVersion" json:"api_version"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Provenance *Provenance `yaml:"provenance,omitempty" json:"provenance,omitempty"`
	Spec       *Spec       `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type Metadata struct {
	// Alpha1-only fields. Alpha2 takes Agent Skills metadata from SKILL.md.
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
	// Alpha2-only distribution and catalog metadata. These fields map to
	// standard/custom OCI annotations; Annotations is an explicit passthrough
	// for additional OCI annotation key/value pairs.
	Title         string            `yaml:"title,omitempty" json:"title,omitempty"`
	Vendor        string            `yaml:"vendor,omitempty" json:"vendor,omitempty"`
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

func Parse(r io.Reader) (*SkillCard, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading skillcard YAML: %w", err)
	}
	var header struct {
		APIVersion string `yaml:"apiVersion"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("parsing skillcard YAML: %w", err)
	}
	if header.APIVersion == APIVersionV1Alpha2 {
		if err := validateAlpha2InputTypes(data); err != nil {
			return nil, fmt.Errorf("parsing skillcard YAML: %w", err)
		}
		var in v1alpha2.SkillCard
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&in); err != nil {
			return nil, fmt.Errorf("parsing skillcard YAML: %w", err)
		}
		return fromV1Alpha2(&in), nil
	}
	if header.APIVersion != APIVersionV1Alpha1 {
		return &SkillCard{APIVersion: header.APIVersion}, nil
	}
	var in v1alpha1.SkillCard
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("parsing skillcard YAML: %w", err)
	}
	return fromV1Alpha1(&in), nil
}

func validateAlpha2InputTypes(data []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	metadataValue, ok := raw["metadata"]
	if !ok {
		return nil
	}
	metadata, ok := metadataValue.(map[string]any)
	if !ok {
		return fmt.Errorf("metadata must be a mapping")
	}
	for _, field := range []string{"version", "title", "vendor", "url", "documentation", "support", "changelog"} {
		if value, exists := metadata[field]; exists {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("metadata.%s must be a string", field)
			}
		}
	}
	if value, exists := metadata["tags"]; exists {
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("metadata.tags must be an array")
		}
		for i, item := range items {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("metadata.tags[%d] must be a string", i)
			}
		}
	}
	if value, exists := metadata["authors"]; exists {
		authors, ok := value.([]any)
		if !ok {
			return fmt.Errorf("metadata.authors must be an array")
		}
		for i, item := range authors {
			author, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("metadata.authors[%d] must be a mapping", i)
			}
			for _, field := range []string{"name", "email"} {
				if fieldValue, exists := author[field]; exists {
					if _, ok := fieldValue.(string); !ok {
						return fmt.Errorf("metadata.authors[%d].%s must be a string", i, field)
					}
				}
			}
		}
	}
	if value, exists := metadata["annotations"]; exists {
		annotations, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("metadata.annotations must be a mapping")
		}
		for key, annotationValue := range annotations {
			if _, ok := annotationValue.(string); !ok {
				return fmt.Errorf("metadata.annotations.%s must be a string", key)
			}
		}
	}
	return nil
}

func Serialize(sc *SkillCard, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	var value any = toV1Alpha1(sc)
	if sc.APIVersion == APIVersionV1Alpha2 {
		value = toV1Alpha2(sc)
	}
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("serializing skillcard: %w", err)
	}
	return enc.Close()
}

func IsDeprecated(sc *SkillCard) bool { return sc.APIVersion == APIVersionV1Alpha1 }

func fromV1Alpha1(in *v1alpha1.SkillCard) *SkillCard {
	sc := &SkillCard{APIVersion: in.APIVersion, Kind: in.Kind}
	sc.Metadata = Metadata{Name: in.Metadata.Name, DisplayName: in.Metadata.DisplayName, Namespace: in.Metadata.Namespace, Version: in.Metadata.Version, Description: in.Metadata.Description, License: in.Metadata.License, Compatibility: in.Metadata.Compatibility, Tags: in.Metadata.Tags, AllowedTools: in.Metadata.AllowedTools}
	for _, a := range in.Metadata.Authors {
		sc.Metadata.Authors = append(sc.Metadata.Authors, Author{Name: a.Name, Email: a.Email})
	}
	if in.Provenance != nil {
		sc.Provenance = &Provenance{Source: in.Provenance.Source, Commit: in.Provenance.Commit, Path: in.Provenance.Path}
	}
	if in.Spec != nil {
		sc.Spec = &Spec{Prompt: in.Spec.Prompt}
		for _, e := range in.Spec.Examples {
			sc.Spec.Examples = append(sc.Spec.Examples, Example{Input: e.Input, Output: e.Output})
		}
		for _, d := range in.Spec.Dependencies {
			sc.Spec.Dependencies = append(sc.Spec.Dependencies, Dependency{Name: d.Name, Version: d.Version})
		}
	}
	return sc
}

func toV1Alpha1(sc *SkillCard) *v1alpha1.SkillCard {
	out := &v1alpha1.SkillCard{APIVersion: sc.APIVersion, Kind: sc.Kind}
	out.Metadata = v1alpha1.Metadata{Name: sc.Metadata.Name, DisplayName: sc.Metadata.DisplayName, Namespace: sc.Metadata.Namespace, Version: sc.Metadata.Version, Description: sc.Metadata.Description, License: sc.Metadata.License, Compatibility: sc.Metadata.Compatibility, Tags: sc.Metadata.Tags, AllowedTools: sc.Metadata.AllowedTools}
	for _, a := range sc.Metadata.Authors {
		out.Metadata.Authors = append(out.Metadata.Authors, v1alpha1.Author{Name: a.Name, Email: a.Email})
	}
	if sc.Provenance != nil {
		out.Provenance = &v1alpha1.Provenance{Source: sc.Provenance.Source, Commit: sc.Provenance.Commit, Path: sc.Provenance.Path}
	}
	if sc.Spec != nil {
		out.Spec = &v1alpha1.Spec{Prompt: sc.Spec.Prompt}
		for _, e := range sc.Spec.Examples {
			out.Spec.Examples = append(out.Spec.Examples, v1alpha1.Example{Input: e.Input, Output: e.Output})
		}
		for _, d := range sc.Spec.Dependencies {
			out.Spec.Dependencies = append(out.Spec.Dependencies, v1alpha1.Dependency{Name: d.Name, Version: d.Version})
		}
	}
	return out
}

func fromV1Alpha2(in *v1alpha2.SkillCard) *SkillCard {
	sc := &SkillCard{APIVersion: in.APIVersion, Kind: in.Kind}
	sc.Metadata = Metadata{Version: in.Metadata.Version, Title: in.Metadata.Title, Vendor: in.Metadata.Vendor, Tags: in.Metadata.Tags, URL: in.Metadata.URL, Documentation: in.Metadata.Documentation, Support: in.Metadata.Support, Changelog: in.Metadata.Changelog, Annotations: in.Metadata.Annotations}
	for _, a := range in.Metadata.Authors {
		sc.Metadata.Authors = append(sc.Metadata.Authors, Author{Name: a.Name, Email: a.Email})
	}
	return sc
}

func toV1Alpha2(sc *SkillCard) *v1alpha2.SkillCard {
	out := &v1alpha2.SkillCard{APIVersion: sc.APIVersion, Kind: sc.Kind}
	out.Metadata = v1alpha2.Metadata{Version: sc.Metadata.Version, Title: sc.Metadata.Title, Vendor: sc.Metadata.Vendor, Tags: sc.Metadata.Tags, URL: sc.Metadata.URL, Documentation: sc.Metadata.Documentation, Support: sc.Metadata.Support, Changelog: sc.Metadata.Changelog, Annotations: sc.Metadata.Annotations}
	for _, a := range sc.Metadata.Authors {
		out.Metadata.Authors = append(out.Metadata.Authors, v1alpha2.Author{Name: a.Name, Email: a.Email})
	}
	return out
}
