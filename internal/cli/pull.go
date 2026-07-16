package cli

import (
	"fmt"
	"strings"

	"github.com/redhat-et/skillimage/pkg/oci"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	var outputDir string
	var tlsVerify bool
	var force bool
	cmd := &cobra.Command{
		Use:   "pull <repository-or-ref>",
		Short: "Pull a skill image from a remote registry",
		Long: `Pull a skill image from an OCI registry to the local store.

Use -o to unpack skill files to a directory. If -o points to an
existing directory, a subdirectory named after the skill is created
automatically.

Examples:
  skillctl pull quay.io/acme/skills/pdf-processing
  skillctl pull quay.io/acme/hr-onboarding:1.0.0
  skillctl pull quay.io/acme/hr-onboarding:1.0.0 -o ./skills/`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(cmd, args[0], outputDir, force, !tlsVerify)
		},
	}
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "unpack skill files to directory")
	cmd.Flags().BoolVar(&force, "force", false, "replace the local managed cursor with remote latest")
	cmd.Flags().BoolVar(&tlsVerify, "tls-verify", true, "require HTTPS and verify certificates")
	return cmd
}

func looksLocal(ref string) bool {
	return !strings.Contains(ref, "/")
}

func runPull(cmd *cobra.Command, ref string, outputDir string, force, skipTLSVerify bool) error {
	if looksLocal(ref) {
		return fmt.Errorf("%s looks like a local reference, not a remote registry\n\nTo install from the local store, use:\n  skillctl install %s --target <agent>\n  skillctl install %s -o <directory>", ref, ref, ref)
	}

	client, err := defaultClient()
	if err != nil {
		return err
	}

	var pulled []string
	desc, err := client.Pull(cmd.Context(), ref, oci.PullOptions{
		OutputDir:     outputDir,
		SkipTLSVerify: skipTLSVerify,
		Force:         force,
		Pulled:        func(pulledRef string) { pulled = append(pulled, pulledRef) },
	})
	if err != nil {
		return fmt.Errorf("pulling: %w", err)
	}

	if len(pulled) == 0 {
		pulled = append(pulled, ref)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Pulled:")
	for _, pulledRef := range pulled {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", pulledRef)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Digest: %s\n", desc.Digest)
	if outputDir != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Unpacked to %s\n", outputDir)
	}
	return nil
}
