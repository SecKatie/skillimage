package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
	"github.com/redhat-et/skillimage/pkg/oci"
	"github.com/redhat-et/skillimage/pkg/skillcard"
)

func newPromoteCmd() *cobra.Command {
	var toState string
	var local bool
	var tlsVerify bool
	cmd := &cobra.Command{
		Use:   "promote <ref>",
		Short: "Promote a skill to the next lifecycle state",
		Long: `Promote a skill image to a new lifecycle state.

State transitions: draft -> testing -> published -> deprecated -> archived

By default, operates on a remote registry. Use --local to promote
images in the local store.

Examples:
  skillctl promote quay.io/acme/hr-onboarding:1.0.0-draft --to testing
  skillctl promote test/test-skill:1.0.0-draft --to testing --local`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromote(cmd, args[0], toState, local, !tlsVerify)
		},
	}
	cmd.Flags().StringVar(&toState, "to", "", "target lifecycle state (required)")
	_ = cmd.MarkFlagRequired("to")
	cmd.Flags().BoolVar(&local, "local", false, "promote in local store instead of remote registry")
	cmd.Flags().BoolVar(&tlsVerify, "tls-verify", true, "require HTTPS and verify certificates")
	return cmd
}

func runPromote(cmd *cobra.Command, ref, toState string, local bool, skipTLSVerify bool) error {
	client, err := defaultClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	var inspected *oci.InspectResult
	if local {
		inspected, err = client.Inspect(ctx, ref)
	} else {
		inspected, err = client.InspectRemote(ctx, ref, oci.InspectOptions{SkipTLSVerify: skipTLSVerify})
	}
	if err != nil {
		return fmt.Errorf("inspecting %s: %w", ref, err)
	}

	if inspected.SkillCardVersion == skillcard.APIVersionV1Alpha2 {
		to, parseErr := lifecycle.ParseStage(toState)
		if parseErr != nil {
			return fmt.Errorf("invalid target stage: %w", parseErr)
		}
		if local {
			err = client.PromoteStageLocal(ctx, ref, to)
		} else {
			err = client.PromoteStage(ctx, ref, to, oci.PromoteOptions{SkipTLSVerify: skipTLSVerify})
		}
	} else {
		to, parseErr := lifecycle.ParseState(toState)
		if parseErr != nil {
			return fmt.Errorf("invalid target state: %w", parseErr)
		}
		if local {
			err = client.PromoteLocal(ctx, ref, to)
		} else {
			err = client.Promote(ctx, ref, to, oci.PromoteOptions{SkipTLSVerify: skipTLSVerify})
		}
	}
	if err != nil {
		return fmt.Errorf("promoting %s: %w", ref, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Promoted %s to %s\n", ref, toState)
	return nil
}

func newDemoteCmd() *cobra.Command {
	var toStage string
	var local bool
	var tlsVerify bool
	cmd := &cobra.Command{
		Use:   "demote <ref>",
		Short: "Demote a v1alpha2 skill to a lower lifecycle stage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			to, err := lifecycle.ParseStage(toStage)
			if err != nil {
				return fmt.Errorf("invalid target stage: %w", err)
			}
			client, err := defaultClient()
			if err != nil {
				return err
			}
			if local {
				err = client.DemoteStageLocal(cmd.Context(), args[0], to)
			} else {
				err = client.DemoteStage(cmd.Context(), args[0], to, oci.PromoteOptions{SkipTLSVerify: !tlsVerify})
			}
			if err != nil {
				return fmt.Errorf("demoting %s: %w", args[0], err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Demoted %s to %s\n", args[0], to)
			return nil
		},
	}
	cmd.Flags().StringVar(&toStage, "to", "", "target lifecycle stage (required)")
	_ = cmd.MarkFlagRequired("to")
	cmd.Flags().BoolVar(&local, "local", false, "demote in local store instead of remote registry")
	cmd.Flags().BoolVar(&tlsVerify, "tls-verify", true, "require HTTPS and verify certificates")
	return cmd
}
