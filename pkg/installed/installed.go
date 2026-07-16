package installed

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/redhat-et/skillimage/pkg/agentskill"
	"github.com/redhat-et/skillimage/pkg/skillcard"
)

// InstalledSkill holds metadata for a skill installed to an agent directory.
type InstalledSkill struct {
	Name    string
	Version string
	Source  string
	Digest  string
	Target  string
	Path    string
}

// Scan reads agent target directories and returns metadata for each
// installed skill. targets maps target names (e.g., "claude") to
// directory paths. Directories that don't exist are silently skipped.
// Malformed skill.yaml files are skipped with a warning to stderr.
func Scan(targets map[string]string) ([]InstalledSkill, error) {
	var skills []InstalledSkill

	for target, dir := range targets {
		receipts := readReceipts(dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", dir, err)
			}
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == ".skillimage" {
				continue
			}

			skillPath := filepath.Join(dir, entry.Name(), "skill.yaml")
			f, err := os.Open(skillPath)
			if err != nil {
				if !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", skillPath, err)
				}
				continue
			}

			sc, err := skillcard.Parse(f)
			_ = f.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", skillPath, err)
				continue
			}

			name := sc.Metadata.Name
			version := sc.Metadata.Version
			if sc.APIVersion == skillcard.APIVersionV1Alpha2 {
				agent, parseErr := agentskill.ParseFile(filepath.Join(dir, entry.Name()))
				if parseErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", skillPath, parseErr)
					continue
				}
				name = agent.Name
			}
			skill := InstalledSkill{
				Name:    name,
				Version: version,
				Target:  target,
				Path:    filepath.Join(dir, entry.Name()),
			}
			if receipt, ok := receipts[name]; ok {
				skill.Version = receipt.Version
				skill.Source = receipt.Source
				skill.Digest = receipt.Digest
			} else if sc.Provenance != nil {
				skill.Source = sc.Provenance.Source
				skill.Digest = sc.Provenance.Commit
			}

			skills = append(skills, skill)
		}
	}

	slices.SortFunc(skills, func(a, b InstalledSkill) int {
		if c := cmp.Compare(a.Target, b.Target); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})

	return skills, nil
}
