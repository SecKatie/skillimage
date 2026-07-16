package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redhat-et/skillimage/pkg/agentskill"
	"github.com/redhat-et/skillimage/pkg/skillcard"
)

type initOptions struct {
	name               string
	version            string
	full               bool
	force              bool
	allowNonconformant bool
}

func newInitCmd() *cobra.Command {
	var opts initOptions
	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize an Agent Skill and v1alpha2 SkillCard",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			return runInit(cmd, dir, opts)
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "skill name when creating SKILL.md (default: directory name)")
	cmd.Flags().StringVar(&opts.version, "version", "0.1.0", "base package version")
	cmd.Flags().BoolVar(&opts.full, "full", false, "create functional scripts, references, and assets examples")
	cmd.Flags().BoolVar(&opts.force, "force", false, "replace files managed by init")
	cmd.Flags().BoolVar(&opts.allowNonconformant, "allow-nonconformant", false, "allow Agent Skills conformance warnings")
	return cmd
}

func runInit(cmd *cobra.Command, dir string, opts initOptions) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	skillPath := filepath.Join(dir, "SKILL.md")
	_, statErr := os.Stat(skillPath)
	hasSkill := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("checking %s: %w", skillPath, statErr)
	}

	name := opts.name
	var generatedSkill []byte
	if hasSkill {
		agent, err := agentskill.ParseFile(dir)
		if err != nil {
			return err
		}
		findings := agentskill.Validate(agent, dir)
		if len(findings) > 0 && !opts.allowNonconformant {
			return fmt.Errorf("agent skill validation failed: %s", joinAgentFindings(findings))
		}
		for _, finding := range findings {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", finding)
		}
	} else {
		if name == "" {
			name = filepath.Base(filepath.Clean(dir))
		}
		generatedSkill = generateSkillMD(name, opts.full)
		agent, err := agentskill.Parse(generatedSkill)
		if err != nil {
			return fmt.Errorf("generating SKILL.md: %w", err)
		}
		if findings := agentskill.Validate(agent, dir); len(findings) > 0 {
			return fmt.Errorf("generated skill would be invalid: %s", joinAgentFindings(findings))
		}
	}

	sc := &skillcard.SkillCard{
		APIVersion: skillcard.APIVersionV1Alpha2,
		Kind:       "SkillCard",
		Metadata:   skillcard.Metadata{Version: opts.version},
	}
	if findings, err := skillcard.Validate(sc); err != nil {
		return err
	} else if len(findings) > 0 {
		return fmt.Errorf("invalid SkillCard: %s", joinCardFindings(findings))
	}
	var card bytes.Buffer
	if err := skillcard.Serialize(sc, &card); err != nil {
		return err
	}

	files := map[string]generatedFile{
		filepath.Join(dir, "skill.yaml"): {data: card.Bytes(), mode: 0o644},
	}
	if !hasSkill {
		files[skillPath] = generatedFile{data: generatedSkill, mode: 0o644}
	}
	if opts.full {
		files[filepath.Join(dir, "scripts", "example.sh")] = generatedFile{data: []byte(exampleScript), mode: 0o755}
		files[filepath.Join(dir, "references", "REFERENCE.md")] = generatedFile{data: []byte(exampleReference), mode: 0o644}
		files[filepath.Join(dir, "assets", "example-template.md")] = generatedFile{data: []byte(exampleAsset), mode: 0o644}
	}

	var collisions []string
	for path := range files {
		if _, err := os.Lstat(path); err == nil {
			collisions = append(collisions, path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("checking %s: %w", path, err)
		}
	}
	sort.Strings(collisions)
	if len(collisions) > 0 && !opts.force {
		return fmt.Errorf("init would overwrite:\n  %s\nrerun with --force to replace these managed files", strings.Join(collisions, "\n  "))
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		file := files[path]
		if info, err := os.Lstat(path); err == nil && info.IsDir() {
			return fmt.Errorf("cannot replace directory %s with a generated file", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("creating parent for %s: %w", path, err)
		}
		if err := os.WriteFile(path, file.data, file.mode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	}
	return nil
}

type generatedFile struct {
	data []byte
	mode os.FileMode
}

func generateSkillMD(name string, full bool) []byte {
	body := "# " + strings.ReplaceAll(name, "-", " ") + "\n\nFollow these instructions to complete tasks related to " + name + ".\n"
	if full {
		body += "\n## Bundled resources\n\n- Run `scripts/example.sh --help` for the example script interface.\n- Read `references/REFERENCE.md` for supporting guidance.\n- Use `assets/example-template.md` as a starter template.\n"
	}
	return []byte(fmt.Sprintf("---\nname: %s\ndescription: Provides a reusable workflow for %s. Use when a task requires this workflow.\n---\n\n%s", name, strings.ReplaceAll(name, "-", " "), body))
}

func joinAgentFindings(findings []agentskill.ValidationError) string {
	parts := make([]string, 0, len(findings))
	for _, finding := range findings {
		parts = append(parts, finding.String())
	}
	return strings.Join(parts, "; ")
}

func joinCardFindings(findings []skillcard.ValidationError) string {
	parts := make([]string, 0, len(findings))
	for _, finding := range findings {
		parts = append(parts, finding.String())
	}
	return strings.Join(parts, "; ")
}

const exampleScript = `#!/bin/sh
set -eu

case "${1:-}" in
  --help|-h)
    echo "Usage: scripts/example.sh [VALUE]"
    echo "Print a deterministic JSON example result."
    exit 0
    ;;
esac

value=${1:-example}
printf '{"ok":true,"value":"%s"}\n' "$value"
`

const exampleReference = `# Reference

Use this file for focused supporting material that should be loaded only when needed.
`

const exampleAsset = `# Example template

Replace this text with reusable output structure or other static skill assets.
`
