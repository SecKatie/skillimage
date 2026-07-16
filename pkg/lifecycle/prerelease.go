package lifecycle

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Stage is the prerelease maturity of a v1alpha2 skill image.
type Stage string

const (
	Alpha Stage = "alpha"
	Beta  Stage = "beta"
	RC    Stage = "rc"
	Final Stage = "final"
)

func ParseStage(value string) (Stage, error) {
	switch Stage(strings.ToLower(value)) {
	case Alpha, Beta, RC, Final:
		return Stage(strings.ToLower(value)), nil
	default:
		return "", fmt.Errorf("invalid prerelease stage %q (expected alpha, beta, rc, or final)", value)
	}
}

func StageRank(stage Stage) int {
	switch stage {
	case Alpha:
		return 0
	case Beta:
		return 1
	case RC:
		return 2
	case Final:
		return 3
	default:
		return -1
	}
}

func EffectiveVersion(base string, stage Stage, number int) (string, error) {
	version, err := semver.StrictNewVersion(base)
	if err != nil || version.Prerelease() != "" || version.Metadata() != "" {
		return "", fmt.Errorf("base version must be a core semantic version: %q", base)
	}
	if stage == Final {
		return base, nil
	}
	if StageRank(stage) < 0 {
		return "", fmt.Errorf("invalid prerelease stage %q", stage)
	}
	if number < 1 {
		return "", fmt.Errorf("prerelease number must be greater than zero")
	}
	return fmt.Sprintf("%s-%s.%d", base, stage, number), nil
}

// ParseEffectiveVersion decomposes a generated alpha2 version. It accepts a
// core semantic version as final and the alpha.N, beta.N, and rc.N forms.
func ParseEffectiveVersion(value string) (base string, stage Stage, number int, err error) {
	v, parseErr := semver.StrictNewVersion(value)
	if parseErr != nil || v.Metadata() != "" {
		return "", "", 0, fmt.Errorf("invalid effective semantic version %q", value)
	}
	base = fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
	if v.Prerelease() == "" {
		return base, Final, 0, nil
	}
	parts := strings.Split(v.Prerelease(), ".")
	if len(parts) != 2 {
		return "", "", 0, fmt.Errorf("unsupported prerelease %q", v.Prerelease())
	}
	stage, parseErr = ParseStage(parts[0])
	if parseErr != nil || stage == Final {
		return "", "", 0, fmt.Errorf("unsupported prerelease %q", v.Prerelease())
	}
	number, parseErr = strconv.Atoi(parts[1])
	if parseErr != nil || number < 1 || strconv.Itoa(number) != parts[1] {
		return "", "", 0, fmt.Errorf("invalid prerelease number %q", parts[1])
	}
	return base, stage, number, nil
}
