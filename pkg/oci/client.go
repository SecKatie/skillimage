package oci

import (
	"oras.land/oras-go/v2/content/oci"

	"github.com/redhat-et/skillimage/pkg/skillcard"
)

// Client provides OCI operations against a local OCI layout store.
type Client struct {
	store     *oci.Store
	storePath string
}

// NewClient creates a new OCI client backed by a local OCI layout store
// at the given path. The directory is created if it does not exist.
func NewClient(storePath string) (*Client, error) {
	store, err := oci.New(storePath)
	if err != nil {
		return nil, err
	}
	return &Client{
		store:     store,
		storePath: storePath,
	}, nil
}

// BuildOptions configures the Build operation.
type BuildOptions struct {
	// Tag is a Podman-style image target. An untagged target selects the
	// repository for managed lifecycle tags; an explicitly tagged target creates
	// only that exact reference. A missing registry is normalized to localhost.
	Tag string
	// Stage overrides alpha2 lifecycle inference.
	Stage string
	// PrereleaseNumber overrides automatic prerelease number allocation.
	PrereleaseNumber int
	// AllowNonconformant downgrades Agent Skills conformance findings to warnings.
	AllowNonconformant bool
	// Warn receives non-fatal diagnostics such as alpha1 deprecation and external
	// symlink dereference notices.
	Warn func(string)
	// Tagged receives each reference created by Build.
	Tagged func(string)
	// MediaType selects the media type profile. Empty or "standard" uses
	// standard OCI types; "redhat" uses Red Hat-specific types for oc-mirror.
	MediaType MediaTypeProfile
	// SkillCard is a pre-built SkillCard to use instead of reading skill.yaml
	// from disk. When non-nil, Build skips the file read/parse/validate steps.
	SkillCard *skillcard.SkillCard
}

// PushOptions configures the Push operation.
type PushOptions struct {
	// SkipTLSVerify disables TLS certificate verification for the
	// remote registry (equivalent to --tls-verify=false).
	SkipTLSVerify bool
	// Force permits intentional replacement of conflicting remote tags.
	Force bool
	// Pushed receives each remote reference updated by Push.
	Pushed func(string)
	// Replaced receives each conflicting tag replaced by a forced push.
	Replaced func(SyncReplacement)
}

// PullOptions configures the Pull operation.
type PullOptions struct {
	// OutputDir, if set, causes the pulled image to be unpacked into this
	// directory after storing it locally.
	OutputDir string
	// SkipTLSVerify disables TLS certificate verification for the
	// remote registry (equivalent to --tls-verify=false).
	SkipTLSVerify bool
	// Force permits local latest to move back to the remote managed cursor.
	Force bool
	// Pulled receives each local reference created by Pull.
	Pulled func(string)
}

// LocalImage holds metadata for an image stored in the local OCI layout.
type LocalImage struct {
	Name    string
	Version string
	Tag     string
	Digest  string
	Status  string
	Created string
}

// PromoteOptions configures the Promote operation.
type PromoteOptions struct {
	// SkipTLSVerify disables TLS certificate verification for the
	// remote registry (equivalent to --tls-verify=false).
	SkipTLSVerify bool
}

// InspectOptions configures the InspectRemote operation.
type InspectOptions struct {
	// SkipTLSVerify disables TLS certificate verification for the
	// remote registry (equivalent to --tls-verify=false).
	SkipTLSVerify bool
}

// InspectResult holds detailed metadata for a skill image.
type InspectResult struct {
	Name             string
	DisplayName      string
	Version          string
	Status           string
	Description      string
	Authors          string
	License          string
	Vendor           string
	URL              string
	Documentation    string
	Tags             string
	Compatibility    string
	AllowedTools     string
	Support          string
	Changelog        string
	SkillCardVersion string
	Annotations      map[string]string
	WordCount        string
	Digest           string
	Created          string
	MediaType        string
	ConfigMediaType  string
	LayerMediaType   string
	Size             int64
	LayerCount       int
}
