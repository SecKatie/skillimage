package skillpackage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/redhat-et/skillimage/pkg/agentskill"
	"github.com/redhat-et/skillimage/pkg/skillcard"
)

type Package struct {
	Dir          string
	Card         *skillcard.SkillCard
	Agent        *agentskill.Skill
	fallbackName string
}

type Options struct {
	Card               *skillcard.SkillCard
	AllowNonconformant bool
	Warn               func(string)
}

func Load(dir string, opts Options) (*Package, error) {
	sc := opts.Card
	if sc == nil {
		f, err := os.Open(filepath.Join(dir, "skill.yaml"))
		if err != nil {
			return nil, fmt.Errorf("opening skill.yaml: %w", err)
		}
		defer func() { _ = f.Close() }()
		var parseErr error
		sc, parseErr = skillcard.Parse(f)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing skill.yaml: %w", parseErr)
		}
	}

	validationErrors, err := skillcard.Validate(sc)
	if err != nil {
		return nil, fmt.Errorf("validating SkillCard: %w", err)
	}
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("SkillCard validation failed: %s", joinCardErrors(validationErrors))
	}

	pkg := &Package{Dir: dir, Card: sc}
	if sc.APIVersion == skillcard.APIVersionV1Alpha1 {
		if opts.Warn != nil {
			opts.Warn("skillimage.io/v1alpha1 is deprecated; see docs/migrations/v1alpha1-to-v1alpha2.md")
		}
		if agent, parseErr := agentskill.ParseFile(dir); parseErr == nil {
			pkg.Agent = agent
		}
		return pkg, nil
	}

	agent, err := agentskill.ParseFile(dir)
	if err != nil {
		return nil, err
	}
	agentErrors := agentskill.Validate(agent, dir)
	if len(agentErrors) > 0 && !opts.AllowNonconformant {
		return nil, fmt.Errorf("agent skill validation failed: %s", joinAgentErrors(agentErrors))
	}
	if opts.AllowNonconformant && opts.Warn != nil {
		for _, finding := range agentErrors {
			opts.Warn(finding.String())
		}
	}
	pkg.Agent = agent
	if opts.AllowNonconformant && !agentskill.HasUsableName(agent, dir) {
		pkg.fallbackName = filepath.Base(filepath.Clean(dir))
	}
	return pkg, nil
}

func (p *Package) Name() string {
	if p.fallbackName != "" {
		return p.fallbackName
	}
	if p.Agent != nil && p.Agent.Name != "" {
		return p.Agent.Name
	}
	if p.Card.Metadata.Name != "" {
		return p.Card.Metadata.Name
	}
	return filepath.Base(filepath.Clean(p.Dir))
}

func (p *Package) Description() string {
	if p.Agent != nil && p.Agent.Description != "" {
		return p.Agent.Description
	}
	return p.Card.Metadata.Description
}

func (p *Package) License() string {
	if p.Agent != nil {
		return p.Agent.License
	}
	return p.Card.Metadata.License
}

func (p *Package) Compatibility() string {
	if p.Agent != nil {
		return p.Agent.Compatibility
	}
	return p.Card.Metadata.Compatibility
}

func (p *Package) AllowedTools() string {
	if p.Agent != nil {
		return p.Agent.AllowedTools
	}
	return p.Card.Metadata.AllowedTools
}

func joinCardErrors(errs []skillcard.ValidationError) string {
	result := ""
	for i, err := range errs {
		if i > 0 {
			result += "; "
		}
		result += err.String()
	}
	return result
}

func joinAgentErrors(errs []agentskill.ValidationError) string {
	result := ""
	for i, err := range errs {
		if i > 0 {
			result += "; "
		}
		result += err.String()
	}
	return result
}
