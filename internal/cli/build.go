package cli

import (
	"fmt"
	"net/url"

	"github.com/redhat-et/skillimage/pkg/oci"
	"github.com/redhat-et/skillimage/pkg/source"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	var tag string
	var mediaType string
	var ref string
	var filter string
	var stage string
	var prereleaseNumber int
	var allowNonconformant bool
	cmd := &cobra.Command{
		Use:   "build <dir-or-url>",
		Short: "Build a skill directory or Git repo into local OCI images",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("prerelease-number") && prereleaseNumber <= 0 {
				return fmt.Errorf("--prerelease-number must be greater than zero")
			}
			if source.IsRemote(args[0]) {
				return runBuildRemote(cmd, args[0], tag, mediaType, ref, filter, stage, prereleaseNumber, allowNonconformant)
			}
			return runBuild(cmd, args[0], tag, mediaType, stage, prereleaseNumber, allowNonconformant)
		},
	}
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "exact image reference to create")
	cmd.Flags().StringVar(&stage, "stage", "", "override lifecycle stage (alpha, beta, rc, final)")
	cmd.Flags().IntVar(&prereleaseNumber, "prerelease-number", 0, "override prerelease number")
	cmd.Flags().BoolVar(&allowNonconformant, "allow-nonconformant", false, "downgrade Agent Skills conformance findings to warnings")
	cmd.Flags().StringVar(&mediaType, "media-type", "", `media type profile: "standard" (default) or "redhat" (for oc-mirror)`)
	cmd.Flags().StringVar(&ref, "ref", "", "Git ref to checkout (branch, tag, or commit SHA)")
	cmd.Flags().StringVar(&filter, "filter", "", "glob pattern to filter skills by name")
	return cmd
}

func runBuild(cmd *cobra.Command, dir, tag, mediaType, stage string, prereleaseNumber int, allowNonconformant bool) error {
	profile, err := oci.ParseMediaTypeProfile(mediaType)
	if err != nil {
		return err
	}

	client, err := defaultClient()
	if err != nil {
		return err
	}

	warn := newWarningPrinter(cmd)
	var refs []string
	desc, err := client.Build(cmd.Context(), dir, oci.BuildOptions{
		Tag:                tag,
		Stage:              stage,
		PrereleaseNumber:   prereleaseNumber,
		AllowNonconformant: allowNonconformant,
		MediaType:          profile,
		Warn:               warn,
		Tagged:             func(ref string) { refs = append(refs, ref) },
	})
	if err != nil {
		return fmt.Errorf("building %s: %w", dir, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Built %s\n", dir)
	for _, ref := range refs {
		fmt.Fprintf(cmd.OutOrStdout(), "Tagged: %s\n", ref)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Digest: %s\n", desc.Digest)
	return nil
}

func runBuildRemote(cmd *cobra.Command, rawURL, tag, mediaType, ref, filter, stage string, prereleaseNumber int, allowNonconformant bool) error {
	profile, err := oci.ParseMediaTypeProfile(mediaType)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	displayURL := sanitizeURL(rawURL)
	fmt.Fprintf(cmd.OutOrStdout(), "Cloning %s", displayURL)
	if ref != "" {
		fmt.Fprintf(cmd.OutOrStdout(), " (ref: %s)", ref)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "...")

	result, err := source.Resolve(ctx, rawURL, ref, filter)
	if err != nil {
		return err
	}
	defer result.Cleanup()

	if tag != "" && len(result.Skills) > 1 {
		return fmt.Errorf("--tag cannot be used when building multiple skills")
	}

	client, err := defaultClient()
	if err != nil {
		return err
	}
	warn := newWarningPrinter(cmd)

	var built, failed int
	for i, skill := range result.Skills {
		fmt.Fprintf(cmd.OutOrStdout(), "Building %s (%d/%d)...\n", skill.Name, i+1, len(result.Skills))

		desc, err := client.Build(ctx, skill.Dir, oci.BuildOptions{
			Tag:                tag,
			Stage:              stage,
			PrereleaseNumber:   prereleaseNumber,
			AllowNonconformant: allowNonconformant,
			MediaType:          profile,
			SkillCard:          skill.SkillCard,
			Warn:               warn,
			Tagged: func(ref string) {
				fmt.Fprintf(cmd.OutOrStdout(), "  Tagged: %s\n", ref)
			},
		})
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  Error: %v\n", err)
			failed++
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  Digest: %s\n", desc.Digest)
		built++
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nBuilt %d skills from %s\n", built, displayURL)
	if failed > 0 {
		return fmt.Errorf("%d skill(s) failed to build", failed)
	}
	return nil
}

func newWarningPrinter(cmd *cobra.Command) func(string) {
	seen := make(map[string]bool)
	return func(message string) {
		if seen[message] {
			return
		}
		seen[message] = true
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", message)
	}
}

func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.User = nil
	return u.String()
}
