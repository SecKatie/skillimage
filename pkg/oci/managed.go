package oci

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
	"github.com/redhat-et/skillimage/pkg/skillcard"
)

// SyncReplacement records an intentional tag replacement.
type SyncReplacement struct {
	Reference string
	OldDigest string
	NewDigest string
}

// SyncResult describes references affected by a managed operation.
type SyncResult struct {
	Repository   string
	Version      string
	VersionRef   string
	LatestRef    string
	Digest       string
	Replacements []SyncReplacement
}

func normalizeManagedRepository(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("repository cannot be empty")
	}
	if _, tag := splitRefTag(ref); tag != "" {
		return "", fmt.Errorf("managed operation requires an untagged repository: %s", ref)
	}
	normalized, err := NormalizeBuildReference(ref)
	if err != nil {
		return "", err
	}
	repository, _ := splitRefTag(normalized)
	return repository, nil
}

// IsManagedReference reports whether ref is an untagged repository operand.
func IsManagedReference(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.Contains(ref, "@") {
		return false
	}
	_, tag := splitRefTag(ref)
	return tag == ""
}

// NormalizeManagedRepository validates and normalizes an untagged repository.
func NormalizeManagedRepository(ref string) (string, error) {
	return normalizeManagedRepository(ref)
}

func compareEffectiveVersions(left, right string) (int, error) {
	l, err := semver.StrictNewVersion(left)
	if err != nil {
		return 0, fmt.Errorf("invalid local effective version %q: %w", left, err)
	}
	r, err := semver.StrictNewVersion(right)
	if err != nil {
		return 0, fmt.Errorf("invalid remote effective version %q: %w", right, err)
	}
	return l.Compare(r), nil
}

func managedResult(repository string, inspected *InspectResult) *SyncResult {
	return &SyncResult{
		Repository: repository,
		Version:    inspected.Version,
		VersionRef: repository + ":" + inspected.Version,
		LatestRef:  repository + ":latest",
		Digest:     inspected.Digest,
	}
}

func pushManaged(ctx context.Context, local, remote oras.Target, repository string, force bool) (*SyncResult, error) {
	return pushManagedWithRemoteRefs(ctx, local, remote, repository, force, func(tag string) string {
		return repository + ":" + tag
	})
}

func pushManagedWithRemoteRefs(ctx context.Context, local, remote oras.Target, repository string, force bool, remoteRef func(string) string) (*SyncResult, error) {
	repository, err := normalizeManagedRepository(repository)
	if err != nil {
		return nil, err
	}
	localLatestRef := repository + ":latest"
	localLatest, err := inspect(ctx, local, localLatestRef)
	if err != nil {
		return nil, fmt.Errorf("inspecting local latest: %w", err)
	}
	if localLatest.SkillCardVersion != skillcard.APIVersionV1Alpha2 {
		return nil, fmt.Errorf("managed push requires a v1alpha2 image")
	}
	result := managedResult(repository, localLatest)
	localVersionDesc, err := local.Resolve(ctx, result.VersionRef)
	if err != nil {
		return nil, fmt.Errorf("resolving local effective version %s: %w", result.VersionRef, err)
	}
	if localVersionDesc.Digest.String() != localLatest.Digest {
		return nil, fmt.Errorf("push refused: local latest and local version tag %s have different content; use an exact push or restore a consistent managed cursor", result.VersionRef)
	}

	remoteLatest, latestErr := inspect(ctx, remote, remoteRef("latest"))
	if latestErr != nil && !errors.Is(latestErr, errdef.ErrNotFound) {
		return nil, fmt.Errorf("inspecting remote latest: %w", latestErr)
	}
	if latestErr == nil {
		comparison, compareErr := compareEffectiveVersions(localLatest.Version, remoteLatest.Version)
		if compareErr != nil {
			return nil, compareErr
		}
		if comparison < 0 && !force {
			return nil, fmt.Errorf("push refused: remote latest is %s, but local latest is %s\n\nSynchronize with the registry:\n  skillctl pull %s\n\nOr replace the remote references intentionally:\n  skillctl push %s --force", remoteLatest.Version, localLatest.Version, repository, repository)
		}
		if comparison == 0 && remoteLatest.Digest != localLatest.Digest && !force {
			return nil, fmt.Errorf("push refused: remote %s has different content\n  remote: %s\n  local:  %s\n\nUse --force to replace the published version", localLatest.Version, remoteLatest.Digest, localLatest.Digest)
		}
		if remoteLatest.Digest != localLatest.Digest && force {
			result.Replacements = append(result.Replacements, SyncReplacement{Reference: result.LatestRef, OldDigest: remoteLatest.Digest, NewDigest: localLatest.Digest})
		}
	}

	remoteVersionRef := remoteRef(localLatest.Version)
	remoteVersionDesc, versionErr := remote.Resolve(ctx, remoteVersionRef)
	if versionErr != nil && !errors.Is(versionErr, errdef.ErrNotFound) {
		return nil, fmt.Errorf("checking remote version %s: %w", result.VersionRef, versionErr)
	}
	if versionErr == nil && remoteVersionDesc.Digest != localVersionDesc.Digest {
		if !force {
			return nil, fmt.Errorf("push refused: remote %s has different content\n  remote: %s\n  local:  %s\n\nUse --force to replace the published version", localLatest.Version, remoteVersionDesc.Digest, localVersionDesc.Digest)
		}
		result.Replacements = append(result.Replacements, SyncReplacement{Reference: result.VersionRef, OldDigest: remoteVersionDesc.Digest.String(), NewDigest: localVersionDesc.Digest.String()})
	}

	if _, err := oras.Copy(ctx, local, result.VersionRef, remote, remoteVersionRef, oras.DefaultCopyOptions); err != nil {
		return nil, fmt.Errorf("pushing %s: %w", result.VersionRef, err)
	}
	if _, err := oras.Copy(ctx, local, localLatestRef, remote, remoteRef("latest"), oras.DefaultCopyOptions); err != nil {
		return nil, fmt.Errorf("pushed %s but failed to update %s: %w; rerun the push to complete publication", result.VersionRef, result.LatestRef, err)
	}
	return result, nil
}

func pullManaged(ctx context.Context, remote, local oras.Target, repository string, force bool) (*SyncResult, error) {
	return pullManagedWithRemoteRefs(ctx, remote, local, repository, force, func(tag string) string {
		return repository + ":" + tag
	})
}

func pullManagedWithRemoteRefs(ctx context.Context, remote, local oras.Target, repository string, force bool, remoteRef func(string) string) (*SyncResult, error) {
	repository, err := normalizeManagedRepository(repository)
	if err != nil {
		return nil, err
	}
	remoteLatest, err := inspect(ctx, remote, remoteRef("latest"))
	if err != nil {
		return nil, fmt.Errorf("inspecting remote latest: %w", err)
	}
	if remoteLatest.SkillCardVersion != skillcard.APIVersionV1Alpha2 {
		return nil, fmt.Errorf("managed pull requires a v1alpha2 image")
	}
	result := managedResult(repository, remoteLatest)
	localLatest, localErr := inspect(ctx, local, result.LatestRef)
	if localErr != nil && !errors.Is(localErr, errdef.ErrNotFound) {
		return nil, fmt.Errorf("inspecting local latest: %w", localErr)
	}
	if localErr == nil {
		comparison, compareErr := compareEffectiveVersions(localLatest.Version, remoteLatest.Version)
		if compareErr != nil {
			return nil, compareErr
		}
		if comparison > 0 && !force {
			return nil, fmt.Errorf("pull refused: local latest is %s, but remote latest is %s; local unpublished progress would be discarded\n\nUse --force to move local latest to the remote version", localLatest.Version, remoteLatest.Version)
		}
		if comparison == 0 && localLatest.Digest != remoteLatest.Digest && !force {
			return nil, fmt.Errorf("pull refused: local and remote %s have different content\n  local:  %s\n  remote: %s\n\nUse --force to accept the remote version", remoteLatest.Version, localLatest.Digest, remoteLatest.Digest)
		}
	}

	desc, err := oras.Copy(ctx, remote, remoteRef("latest"), local, result.LatestRef, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("pulling %s: %w", result.LatestRef, err)
	}
	if err := local.Tag(ctx, desc, result.VersionRef); err != nil {
		return nil, fmt.Errorf("tagging pulled image as %s: %w", result.VersionRef, err)
	}
	result.Digest = desc.Digest.String()
	return result, nil
}

// PromoteManagedLocal advances a managed repository's local latest cursor.
// An empty target advances exactly one stage.
func (c *Client) PromoteManagedLocal(ctx context.Context, repository, target string) (*SyncResult, error) {
	repository, err := normalizeManagedRepository(repository)
	if err != nil {
		return nil, err
	}
	latestRef := repository + ":latest"
	current, err := c.Inspect(ctx, latestRef)
	if err != nil {
		return nil, fmt.Errorf("inspecting local latest: %w", err)
	}
	if current.SkillCardVersion != skillcard.APIVersionV1Alpha2 {
		return nil, fmt.Errorf("managed promotion requires a v1alpha2 image")
	}
	base, from, _, err := lifecycle.ParseEffectiveVersion(current.Version)
	if err != nil {
		return nil, err
	}
	var to lifecycle.Stage
	if target != "" {
		to, err = lifecycle.ParseStage(target)
		if err != nil {
			return nil, err
		}
	} else {
		switch from {
		case lifecycle.Alpha:
			to = lifecycle.Beta
		case lifecycle.Beta:
			to = lifecycle.RC
		case lifecycle.RC:
			to = lifecycle.Final
		case lifecycle.Final:
			return nil, fmt.Errorf("%s is already final at base version %s; bump metadata.version before promoting again", latestRef, base)
		default:
			return nil, fmt.Errorf("unsupported lifecycle stage %q", from)
		}
	}
	if err := c.PromoteStageLocal(ctx, latestRef, to); err != nil {
		return nil, err
	}
	promoted, err := c.Inspect(ctx, latestRef)
	if err != nil {
		return nil, err
	}
	return managedResult(repository, promoted), nil
}

// DemoteManagedLocal moves a managed repository's local latest cursor to a
// lower lifecycle stage.
func (c *Client) DemoteManagedLocal(ctx context.Context, repository, target string) (*SyncResult, error) {
	repository, err := normalizeManagedRepository(repository)
	if err != nil {
		return nil, err
	}
	to, err := lifecycle.ParseStage(target)
	if err != nil {
		return nil, err
	}
	latestRef := repository + ":latest"
	current, err := c.Inspect(ctx, latestRef)
	if err != nil {
		return nil, fmt.Errorf("inspecting local latest: %w", err)
	}
	if current.SkillCardVersion != skillcard.APIVersionV1Alpha2 {
		return nil, fmt.Errorf("managed demotion requires a v1alpha2 image")
	}
	if err := c.DemoteStageLocal(ctx, latestRef, to); err != nil {
		return nil, err
	}
	demoted, err := c.Inspect(ctx, latestRef)
	if err != nil {
		return nil, err
	}
	return managedResult(repository, demoted), nil
}
