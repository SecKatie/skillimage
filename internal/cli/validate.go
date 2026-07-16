package cli

import (
	"fmt"
	"os"

	"github.com/redhat-et/skillimage/pkg/skillcard"
	"github.com/redhat-et/skillimage/pkg/skillpackage"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var allowNonconformant bool
	cmd := &cobra.Command{
		Use:   "validate <dir|file>",
		Short: "Validate a SkillCard against the JSON Schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd, args, allowNonconformant)
		},
	}
	cmd.Flags().BoolVar(&allowNonconformant, "allow-nonconformant", false, "downgrade Agent Skills conformance findings to warnings")
	return cmd
}

func runValidate(cmd *cobra.Command, args []string, allowNonconformant bool) error {
	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("accessing %s: %w", path, err)
	}
	if info.IsDir() {
		_, err := skillpackage.Load(path, skillpackage.Options{
			AllowNonconformant: allowNonconformant,
			Warn:               newWarningPrinter(cmd),
		})
		if err != nil {
			return fmt.Errorf("validating %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s is valid\n", path)
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	sc, err := skillcard.Parse(f)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	if skillcard.IsDeprecated(sc) {
		newWarningPrinter(cmd)("skillimage.io/v1alpha1 is deprecated; see docs/migrations/v1alpha1-to-v1alpha2.md")
	}

	errs, err := skillcard.Validate(sc)
	if err != nil {
		return fmt.Errorf("validating %s: %w", path, err)
	}

	if len(errs) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ %s has %d error(s):\n", path, len(errs))
		for _, e := range errs {
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s: %s\n", e.Field, e.Message)
		}
		return fmt.Errorf("validation failed with %d error(s)", len(errs))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ %s is valid\n", path)
	return nil
}
