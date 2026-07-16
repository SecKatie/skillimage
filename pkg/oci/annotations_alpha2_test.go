package oci

import "testing"

func TestSPDXExpressionValidation(t *testing.T) {
	for _, value := range []string{"MIT", "Apache-2.0", "Unlicense", "MIT OR Apache-2.0", "GPL-2.0-only WITH Classpath-exception-2.0"} {
		if !isSPDXExpression(value) {
			t.Errorf("expected valid SPDX expression %q", value)
		}
	}
	for _, value := range []string{"", "LICENSE.txt", "see the bundled license file", "Proprietary"} {
		if isSPDXExpression(value) {
			t.Errorf("expected non-SPDX value %q to be rejected", value)
		}
	}
}
