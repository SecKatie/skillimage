package cli

import (
	"fmt"

	"github.com/redhat-et/skillimage/pkg/oci"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var tlsVerify bool
	var force bool
	cmd := &cobra.Command{
		Use:   "push <repository-or-ref>",
		Short: "Push a skill image from local store to a remote registry",
		Long: `Push a skill image to a remote OCI registry.

An untagged repository publishes the effective version referenced by local
latest, then moves remote latest. An explicitly tagged reference pushes only
that tag.

Examples:
  skillctl push quay.io/acme/skills/pdf-processing
  skillctl push quay.io/acme/skills/pdf-processing:canary`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd, args[0], force, !tlsVerify)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "replace conflicting remote tags intentionally")
	cmd.Flags().BoolVar(&tlsVerify, "tls-verify", true, "require HTTPS and verify certificates")
	return cmd
}

func runPush(cmd *cobra.Command, ref string, force, skipTLSVerify bool) error {
	client, err := defaultClient()
	if err != nil {
		return err
	}
	return pushWithClient(cmd, client, ref, force, skipTLSVerify)
}

func pushWithClient(cmd *cobra.Command, client *oci.Client, ref string, force, skipTLSVerify bool) error {
	var pushed []string
	err := client.Push(cmd.Context(), ref, oci.PushOptions{
		SkipTLSVerify: skipTLSVerify,
		Force:         force,
		Pushed:        func(pushedRef string) { pushed = append(pushed, pushedRef) },
		Replaced: func(replacement oci.SyncReplacement) {
			fmt.Fprintf(cmd.OutOrStdout(), "Replacing %s\n  old: %s\n  new: %s\n", replacement.Reference, replacement.OldDigest, replacement.NewDigest)
		},
	})
	if err != nil {
		return fmt.Errorf("pushing: %w", err)
	}
	if len(pushed) == 0 {
		pushed = append(pushed, ref)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Published:")
	for _, pushedRef := range pushed {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", pushedRef)
	}
	return nil
}
