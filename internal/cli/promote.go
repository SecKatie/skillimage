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
	var push bool
	var force bool
	var tlsVerify bool
	cmd := &cobra.Command{
		Use:   "promote <ref>",
		Short: "Promote a skill to a higher lifecycle stage",
		Long: `Promote a local skill image to a higher lifecycle stage.

v1alpha2 lifecycle: alpha -> beta -> rc -> final
v1alpha1 compatibility: draft -> testing -> published -> deprecated -> archived

An untagged repository uses local latest and automatically advances one stage.
Use --to to skip intentionally. Promotion remains local unless --push is used.

Examples:
  skillctl promote bumbleforge.com/kglitchy/skills/feature-brainstorming
  skillctl promote ghcr.io/acme/skills/pdf-processing --to rc
  skillctl promote ghcr.io/acme/skills/pdf-processing --push`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if force && !push {
				return fmt.Errorf("--force requires --push")
			}
			return runPromote(cmd, args[0], toState, local, push, force, !tlsVerify)
		},
	}
	cmd.Flags().StringVar(&toState, "to", "", "override target lifecycle stage or legacy state")
	cmd.Flags().BoolVar(&local, "local", false, "deprecated: promotion is always local")
	cmd.Flags().BoolVar(&push, "push", false, "publish the promoted version and latest")
	cmd.Flags().BoolVar(&force, "force", false, "replace conflicting remote tags when used with --push")
	cmd.Flags().BoolVar(&tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when used with --push")
	return cmd
}

func runPromote(cmd *cobra.Command, ref, toState string, local, push, force, skipTLSVerify bool) error {
	client, err := defaultClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if local {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: --local is deprecated; promotion is local by default")
	}

	if oci.IsManagedReference(ref) {
		result, promoteErr := client.PromoteManagedLocal(ctx, ref, toState)
		if promoteErr != nil {
			return fmt.Errorf("promoting %s: %w", ref, promoteErr)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Promoted local latest to %s\nLocal:\n  %s\n  %s\n", result.Version, result.VersionRef, result.LatestRef)
		if push {
			if err := pushWithClient(cmd, client, result.Repository, force, skipTLSVerify); err != nil {
				return fmt.Errorf("promotion succeeded locally but publication failed: %w", err)
			}
		}
		return nil
	}
	if push {
		return fmt.Errorf("--push requires an untagged managed repository")
	}

	inspected, err := client.Inspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("inspecting local %s: %w", ref, err)
	}

	promotedTo := toState
	if inspected.SkillCardVersion == skillcard.APIVersionV1Alpha2 {
		to, parseErr := targetPromotionStage(inspected.Status, toState)
		if parseErr != nil {
			return fmt.Errorf("invalid target stage: %w", parseErr)
		}
		promotedTo = string(to)
		err = client.PromoteStageLocal(ctx, ref, to)
	} else {
		if toState == "" {
			return fmt.Errorf("--to is required for legacy v1alpha1 promotion")
		}
		to, parseErr := lifecycle.ParseState(toState)
		if parseErr != nil {
			return fmt.Errorf("invalid target state: %w", parseErr)
		}
		err = client.PromoteLocal(ctx, ref, to)
	}
	if err != nil {
		return fmt.Errorf("promoting %s: %w", ref, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Promoted %s to %s\n", ref, promotedTo)
	return nil
}

func targetPromotionStage(current, override string) (lifecycle.Stage, error) {
	if override != "" {
		return lifecycle.ParseStage(override)
	}
	from, err := lifecycle.ParseStage(current)
	if err != nil {
		return "", err
	}
	switch from {
	case lifecycle.Alpha:
		return lifecycle.Beta, nil
	case lifecycle.Beta:
		return lifecycle.RC, nil
	case lifecycle.RC:
		return lifecycle.Final, nil
	case lifecycle.Final:
		return "", fmt.Errorf("already final; bump metadata.version before promoting again")
	default:
		return "", fmt.Errorf("unsupported lifecycle stage %q", from)
	}
}

func newDemoteCmd() *cobra.Command {
	var toStage string
	var local bool
	cmd := &cobra.Command{
		Use:   "demote <ref>",
		Short: "Demote a v1alpha2 skill to a lower lifecycle stage",
		Long: `Demote a v1alpha2 skill image to a lower lifecycle stage.

Stage order: alpha < beta < rc < final

An untagged repository operates on local latest. Demotion preserves historical
version tags and moves the local lifecycle cursor.

Examples:
  skillctl demote bumbleforge.com/kglitchy/skills/feature-brainstorming --to beta
  skillctl demote ghcr.io/acme/skills/pdf-processing:1.2.0-rc.1 --to beta`,
		Args: cobra.ExactArgs(1),
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
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: --local is deprecated; demotion is local by default")
			}
			if oci.IsManagedReference(args[0]) {
				result, demoteErr := client.DemoteManagedLocal(cmd.Context(), args[0], toStage)
				if demoteErr != nil {
					return fmt.Errorf("demoting %s: %w", args[0], demoteErr)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Demoted local latest to %s\nLocal:\n  %s\n  %s\n", result.Version, result.VersionRef, result.LatestRef)
				return nil
			}
			if err = client.DemoteStageLocal(cmd.Context(), args[0], to); err != nil {
				return fmt.Errorf("demoting %s: %w", args[0], err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Demoted %s to %s\n", args[0], to)
			return nil
		},
	}
	cmd.Flags().StringVar(&toStage, "to", "", "target lifecycle stage (required)")
	_ = cmd.MarkFlagRequired("to")
	cmd.Flags().BoolVar(&local, "local", false, "deprecated: demotion is always local")
	return cmd
}
