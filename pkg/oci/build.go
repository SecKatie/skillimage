package oci

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	_ "crypto/sha256" // Register SHA256 algorithm for go-digest.
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
	"github.com/redhat-et/skillimage/pkg/skillcard"
	"github.com/redhat-et/skillimage/pkg/skillpackage"
)

// Build reads a skill directory, validates the SkillCard, creates an OCI image,
// and stores it in the local OCI layout. It returns the manifest descriptor.
func (c *Client) Build(ctx context.Context, skillDir string, opts BuildOptions) (ocispec.Descriptor, error) {
	if opts.PrereleaseNumber < 0 {
		return ocispec.Descriptor{}, fmt.Errorf("prerelease number must be greater than zero")
	}
	pkg, err := skillpackage.Load(skillDir, skillpackage.Options{
		Card:               opts.SkillCard,
		AllowNonconformant: opts.AllowNonconformant,
		Warn:               opts.Warn,
	})
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	sc := pkg.Card

	// 2b. Count words in SKILL.md if present, excluding YAML frontmatter.
	var wordCount int
	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	if sc.APIVersion == skillcard.APIVersionV1Alpha2 && pkg.Agent != nil {
		wordCount = len(strings.Fields(pkg.Agent.Body))
	} else if data, err := os.ReadFile(skillMDPath); err == nil {
		wordCount = len(strings.Fields(stripFrontmatter(string(data))))
	}

	// 3. Resolve media types for the selected profile.
	layerMediaType, configMediaType := resolveMediaTypes(opts.MediaType)

	// 4. Create a tar.gz layer of all files in the directory.
	var layerBuf *bytes.Buffer
	var uncompressedDigest godigest.Digest
	if sc.APIVersion == skillcard.APIVersionV1Alpha2 {
		layerBuf, uncompressedDigest, err = createAlpha2Layer(skillDir, opts.Warn)
	} else {
		layerBuf, uncompressedDigest, err = createLayer(skillDir)
	}
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("creating layer: %w", err)
	}

	// 5. Push the layer blob to the store.
	layerBytes := layerBuf.Bytes()
	layerDigest := godigest.FromBytes(layerBytes)
	layerDesc := ocispec.Descriptor{
		MediaType: layerMediaType,
		Digest:    layerDigest,
		Size:      int64(len(layerBytes)),
	}
	if err := c.store.Push(ctx, layerDesc, bytes.NewReader(layerBytes)); err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return ocispec.Descriptor{}, fmt.Errorf("pushing layer: %w", err)
	}

	// 6. Create and push the OCI image config.
	configBytes, err := buildImageConfig(uncompressedDigest)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("building image config: %w", err)
	}
	configDigest := godigest.FromBytes(configBytes)
	configDesc := ocispec.Descriptor{
		MediaType: configMediaType,
		Digest:    configDigest,
		Size:      int64(len(configBytes)),
	}
	if err := c.store.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return ocispec.Descriptor{}, fmt.Errorf("pushing config: %w", err)
	}

	// 7. Build annotations from SkillCard.
	if sc.APIVersion == skillcard.APIVersionV1Alpha1 {
		annotations := buildAnnotations(sc, wordCount)
		tag := opts.Tag
		if tag == "" {
			tag = lifecycle.TagForState(sc.Metadata.Version, lifecycle.Draft)
		}
		ref := fmt.Sprintf("%s/%s:%s", sc.Metadata.Namespace, sc.Metadata.Name, tag)
		desc, buildErr := c.buildAndTagManifest(ctx, configDesc, []ocispec.Descriptor{layerDesc}, annotations, "", ref)
		if buildErr == nil && opts.Tagged != nil {
			opts.Tagged(ref)
		}
		return desc, buildErr
	}

	return c.buildAlpha2Manifest(ctx, pkg, configDesc, layerDesc, wordCount, opts)
}

func (c *Client) buildAlpha2Manifest(ctx context.Context, pkg *skillpackage.Package, configDesc, layerDesc ocispec.Descriptor, wordCount int, opts BuildOptions) (ocispec.Descriptor, error) {
	repository := "localhost/" + pkg.Name()
	exactRef := ""
	if opts.Tag != "" {
		var err error
		exactRef, err = NormalizeBuildReference(opts.Tag)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		repository, _ = splitRefTag(exactRef)
	}

	stage, number, err := c.nextAlpha2Version(ctx, repository, pkg.Card.Metadata.Version, opts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	effectiveVersion, err := lifecycle.EffectiveVersion(pkg.Card.Metadata.Version, stage, number)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	annotations := buildAlpha2Annotations(pkg, effectiveVersion, stage, wordCount)
	if pkg.License() != "" && annotations[ocispec.AnnotationLicenses] == "" && opts.Warn != nil {
		opts.Warn("SKILL.md license is not a valid SPDX expression; preserving it in package content without an OCI license annotation")
	}

	ref := exactRef
	if ref == "" {
		ref = repository + ":" + effectiveVersion
		if _, resolveErr := c.store.Resolve(ctx, ref); resolveErr == nil {
			return ocispec.Descriptor{}, fmt.Errorf("immutable version tag %s already exists", ref)
		}
	}
	desc, err := c.buildAndTagManifest(ctx, configDesc, []ocispec.Descriptor{layerDesc}, annotations, "", ref)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	if opts.Tagged != nil {
		opts.Tagged(ref)
	}
	if exactRef == "" {
		latest := repository + ":latest"
		if err := c.store.Tag(ctx, desc, latest); err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("tagging manifest as %s: %w", latest, err)
		}
		if opts.Tagged != nil {
			opts.Tagged(latest)
		}
	}
	return desc, nil
}

func (c *Client) nextAlpha2Version(ctx context.Context, repository, base string, opts BuildOptions) (lifecycle.Stage, int, error) {
	stage := lifecycle.Alpha
	number := 1
	latest := repository + ":latest"
	if result, err := c.Inspect(ctx, latest); err == nil && result.SkillCardVersion == skillcard.APIVersionV1Alpha2 {
		latestBase, latestStage, latestNumber, parseErr := lifecycle.ParseEffectiveVersion(result.Version)
		if parseErr == nil && latestBase == base {
			if latestStage == lifecycle.Final && opts.Stage == "" {
				return "", 0, fmt.Errorf("%s is already final at base version %s; bump metadata.version before building again", latest, base)
			}
			stage, number = latestStage, latestNumber+1
		}
	}
	if opts.Stage != "" {
		var err error
		stage, err = lifecycle.ParseStage(opts.Stage)
		if err != nil {
			return "", 0, err
		}
		number = 1
	}
	if stage == lifecycle.Final {
		number = 0
	} else if opts.PrereleaseNumber > 0 {
		number = opts.PrereleaseNumber
	} else {
		for {
			version, _ := lifecycle.EffectiveVersion(base, stage, number)
			if _, err := c.store.Resolve(ctx, repository+":"+version); err != nil {
				break
			}
			number++
		}
	}
	if opts.PrereleaseNumber > 0 && stage == lifecycle.Final {
		return "", 0, fmt.Errorf("--prerelease-number cannot be used with final stage")
	}
	return stage, number, nil
}

// NormalizeBuildReference applies Podman-compatible short-name defaults for a
// build target. Build targets must be tags, never digests.
func NormalizeBuildReference(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("image reference cannot be empty")
	}
	if strings.Contains(ref, "@") {
		return "", fmt.Errorf("digest references are not valid build targets: %s", ref)
	}
	repo, tag := splitRefTag(ref)
	if tag == "" {
		tag = "latest"
	}
	first := repo
	if idx := strings.IndexByte(repo, '/'); idx >= 0 {
		first = repo[:idx]
	}
	if !strings.Contains(first, ".") && !strings.Contains(first, ":") && first != "localhost" {
		repo = "localhost/" + repo
	}
	if strings.HasPrefix(repo, "/") || strings.HasSuffix(repo, "/") || strings.Contains(repo, "//") {
		return "", fmt.Errorf("invalid image reference %q", ref)
	}
	if tag == "" || strings.ContainsAny(tag, "/@") || strings.Contains(tag, " ") {
		return "", fmt.Errorf("invalid image tag in %q", ref)
	}
	normalized := repo + ":" + tag
	if _, err := registry.ParseReference(normalized); err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", ref, err)
	}
	return normalized, nil
}

// ListLocal reads the store's tags and returns image metadata from manifest annotations.
func (c *Client) ListLocal() ([]LocalImage, error) {
	images, err := c.listLocal(context.Background())
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	return images, nil
}

func (c *Client) listLocal(ctx context.Context) ([]LocalImage, error) {
	var images []LocalImage

	err := c.store.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			desc, err := c.store.Resolve(ctx, tag)
			if err != nil {
				continue
			}

			img, err := c.imageFromManifest(ctx, tag, desc)
			if err != nil {
				continue
			}
			images = append(images, *img)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return images, nil
}

// imageFromManifest fetches a manifest and extracts LocalImage metadata.
func (c *Client) imageFromManifest(ctx context.Context, tag string, desc ocispec.Descriptor) (*LocalImage, error) {
	rc, err := c.store.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	manifestBytes, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, err
	}

	ann := manifest.Annotations
	if ann == nil {
		return nil, fmt.Errorf("no annotations")
	}

	name := parseNameFromTag(tag)
	_, tagPart := splitRefTag(tag)

	return &LocalImage{
		Name:    name,
		Version: ann[ocispec.AnnotationVersion],
		Tag:     tagPart,
		Digest:  desc.Digest.String(),
		Status:  ann[lifecycle.StatusAnnotation],
		Created: ann[ocispec.AnnotationCreated],
	}, nil
}

// parseNameFromTag extracts the "namespace/name" portion from a tag reference
// like "namespace/name:tag".
func parseNameFromTag(ref string) string {
	if idx := strings.Index(ref, "@"); idx >= 0 {
		return ref[:idx]
	}
	if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		return ref[:idx]
	}
	return ref
}

// createLayer builds a tar.gz archive of all files in dir, skipping hidden
// directories. It returns the compressed buffer and the digest of the
// uncompressed tar (for diff_ids in the image config).
func createLayer(dir string) (*bytes.Buffer, godigest.Digest, error) {
	// First, build the uncompressed tar to compute its digest.
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Skip the root directory entry itself.
		if rel == "." {
			return nil
		}

		// Skip hidden directories.
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip hidden files.
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		if !info.Mode().IsRegular() && !info.IsDir() {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating tar header for %s: %w", rel, err)
		}
		header.Name = filepath.ToSlash(rel)

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for %s: %w", rel, err)
		}

		if info.Mode().IsRegular() {
			if err := copyFileToTar(tw, path, rel); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("walking skill directory: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, "", fmt.Errorf("closing tar writer: %w", err)
	}

	// Compute digest of uncompressed tar for diff_ids.
	uncompressedDigest := godigest.FromBytes(tarBuf.Bytes())

	// Now gzip the tar.
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if _, err := io.Copy(gw, &tarBuf); err != nil {
		return nil, "", fmt.Errorf("compressing layer: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, "", fmt.Errorf("closing gzip writer: %w", err)
	}

	return &gzBuf, uncompressedDigest, nil
}

// createAlpha2Layer archives the complete Agent Skill package, including
// dotfiles. Symlinks are followed intentionally and their targets are copied
// into the archive as regular files/directories.
func createAlpha2Layer(dir string, warn func(string)) (*bytes.Buffer, godigest.Digest, error) {
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, "", fmt.Errorf("resolving skill directory %s: %w", dir, err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, "", fmt.Errorf("resolving absolute skill directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, "", fmt.Errorf("reading skill directory: %w", err)
	}
	if !info.IsDir() {
		return nil, "", fmt.Errorf("skill path %s is not a directory", dir)
	}

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	activeDirs := make(map[string]bool)
	activeDirs[root] = true

	var add func(actual, rel string) error
	add = func(actual, rel string) error {
		entryInfo, err := os.Lstat(actual)
		if err != nil {
			return fmt.Errorf("reading %s: %w", rel, err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			target, resolveErr := filepath.EvalSymlinks(actual)
			if resolveErr != nil {
				return fmt.Errorf("resolving symlink %s: %w", rel, resolveErr)
			}
			target, resolveErr = filepath.Abs(target)
			if resolveErr != nil {
				return fmt.Errorf("resolving symlink %s: %w", rel, resolveErr)
			}
			if !pathWithin(root, target) && warn != nil {
				warn(fmt.Sprintf("dereferencing external symlink %s -> %s", rel, target))
			}
			actual = target
			entryInfo, err = os.Stat(target)
			if err != nil {
				return fmt.Errorf("reading symlink target for %s: %w", rel, err)
			}
		}

		if !entryInfo.Mode().IsRegular() && !entryInfo.IsDir() {
			return fmt.Errorf("unsupported special file %s (%s)", rel, entryInfo.Mode().Type())
		}
		if entryInfo.IsDir() {
			canonical, resolveErr := filepath.EvalSymlinks(actual)
			if resolveErr != nil {
				return fmt.Errorf("resolving directory %s: %w", rel, resolveErr)
			}
			if activeDirs[canonical] {
				return fmt.Errorf("symlink cycle detected at %s", rel)
			}
			activeDirs[canonical] = true
			defer delete(activeDirs, canonical)
		}

		header, err := tar.FileInfoHeader(entryInfo, "")
		if err != nil {
			return fmt.Errorf("creating tar header for %s: %w", rel, err)
		}
		header.Name = filepath.ToSlash(rel)
		if entryInfo.IsDir() && !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""
		header.ModTime = time.Unix(0, 0).UTC()
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for %s: %w", rel, err)
		}
		if entryInfo.Mode().IsRegular() {
			return copyFileToTar(tw, actual, rel)
		}

		entries, err := os.ReadDir(actual)
		if err != nil {
			return fmt.Errorf("reading directory %s: %w", rel, err)
		}
		for _, entry := range entries {
			if err := add(filepath.Join(actual, entry.Name()), filepath.Join(rel, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, "", fmt.Errorf("reading skill directory: %w", err)
	}
	for _, entry := range entries {
		if err := add(filepath.Join(root, entry.Name()), entry.Name()); err != nil {
			_ = tw.Close()
			return nil, "", err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, "", fmt.Errorf("closing tar writer: %w", err)
	}

	uncompressedDigest := godigest.FromBytes(tarBuf.Bytes())
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.ModTime = time.Unix(0, 0).UTC()
	if _, err := io.Copy(gw, &tarBuf); err != nil {
		return nil, "", fmt.Errorf("compressing layer: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, "", fmt.Errorf("closing gzip writer: %w", err)
	}
	return &gzBuf, uncompressedDigest, nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func copyFileToTar(tw *tar.Writer, path, rel string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", rel, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("copying %s: %w", rel, err)
	}
	return nil
}

// stripFrontmatter removes YAML frontmatter (delimited by "---") from
// SKILL.md content so that metadata fields don't inflate the word count.
func stripFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---") {
		return s
	}
	end := strings.Index(s[3:], "\n---")
	if end < 0 {
		return s
	}
	// Skip past the closing "---" and its newline.
	body := s[3+end+4:]
	return body
}

// buildImageConfig creates the OCI image config JSON (FROM scratch equivalent).
func buildImageConfig(diffID godigest.Digest) ([]byte, error) {
	config := ocispec.Image{
		Platform: ocispec.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
		RootFS: ocispec.RootFS{
			Type:    "layers",
			DiffIDs: []godigest.Digest{diffID},
		},
	}
	return json.Marshal(config)
}
