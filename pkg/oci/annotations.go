package oci

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/github/go-spdx/v2/spdxexp"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
	"github.com/redhat-et/skillimage/pkg/skillcard"
	"github.com/redhat-et/skillimage/pkg/skillpackage"
)

// Custom annotation keys for catalog metadata.
const (
	AnnotationTags             = "io.skillimage.tags"
	AnnotationCompatibility    = "io.skillimage.compatibility"
	AnnotationWordCount        = "io.skillimage.wordcount"
	AnnotationAllowedTools     = "io.skillimage.allowed-tools"
	AnnotationSupport          = "io.skillimage.support"
	AnnotationChangelog        = "io.skillimage.changelog"
	AnnotationSkillCardVersion = "io.skillimage.skillcard-version"
)

// buildAnnotations maps SkillCard fields to standard OCI annotation keys
// and the custom skillimage status annotation.
func buildAnnotations(sc *skillcard.SkillCard, wordCount int) map[string]string {
	ann := make(map[string]string)

	// Title: use display-name if set, otherwise name.
	title := sc.Metadata.DisplayName
	if title == "" {
		title = sc.Metadata.Name
	}
	ann[ocispec.AnnotationTitle] = title

	// Description: first 256 characters.
	desc := sc.Metadata.Description
	if len(desc) > 256 {
		desc = desc[:256]
		for len(desc) > 0 && !utf8.ValidString(desc) {
			desc = desc[:len(desc)-1]
		}
	}
	ann[ocispec.AnnotationDescription] = desc

	// Version.
	ann[ocispec.AnnotationVersion] = sc.Metadata.Version

	// Authors: comma-separated "name <email>".
	if len(sc.Metadata.Authors) > 0 {
		var parts []string
		for _, a := range sc.Metadata.Authors {
			if a.Email != "" {
				parts = append(parts, fmt.Sprintf("%s <%s>", a.Name, a.Email))
			} else {
				parts = append(parts, a.Name)
			}
		}
		ann[ocispec.AnnotationAuthors] = strings.Join(parts, ", ")
	}

	// License.
	if sc.Metadata.License != "" {
		ann[ocispec.AnnotationLicenses] = sc.Metadata.License
	}

	// Vendor: namespace.
	ann[ocispec.AnnotationVendor] = sc.Metadata.Namespace

	// Created: RFC 3339 timestamp.
	ann[ocispec.AnnotationCreated] = time.Now().UTC().Format(time.RFC3339)

	// Provenance fields.
	if sc.Provenance != nil {
		if sc.Provenance.Source != "" {
			ann[ocispec.AnnotationSource] = sc.Provenance.Source
		}
		if sc.Provenance.Commit != "" {
			ann[ocispec.AnnotationRevision] = sc.Provenance.Commit
		}
	}

	// Lifecycle status: initial state is always draft.
	ann[lifecycle.StatusAnnotation] = string(lifecycle.Draft)

	// Tags: JSON-encoded string array. Marshal of []string cannot fail in practice.
	if len(sc.Metadata.Tags) > 0 {
		tagsJSON, err := json.Marshal(sc.Metadata.Tags)
		if err == nil {
			ann[AnnotationTags] = string(tagsJSON)
		}
	}

	// Compatibility.
	if sc.Metadata.Compatibility != "" {
		ann[AnnotationCompatibility] = sc.Metadata.Compatibility
	}

	// Word count of SKILL.md.
	if wordCount > 0 {
		ann[AnnotationWordCount] = strconv.Itoa(wordCount)
	}

	return ann
}

// buildAlpha2Annotations maps the normalized package, whose SKILL.md is
// authoritative, to OCI catalog annotations.
func buildAlpha2Annotations(pkg *skillpackage.Package, effectiveVersion string, stage lifecycle.Stage, wordCount int) map[string]string {
	ann := make(map[string]string)
	title := pkg.Card.Metadata.Title
	if title == "" {
		title = pkg.Name()
	}
	ann[ocispec.AnnotationTitle] = title
	ann[ocispec.AnnotationDescription] = truncateAnnotation(pkg.Description(), 256)
	ann[ocispec.AnnotationVersion] = effectiveVersion
	ann[ocispec.AnnotationCreated] = time.Now().UTC().Format(time.RFC3339)
	ann[lifecycle.StatusAnnotation] = string(stage)
	ann[AnnotationSkillCardVersion] = skillcard.APIVersionV1Alpha2

	if len(pkg.Card.Metadata.Authors) > 0 {
		var parts []string
		for _, author := range pkg.Card.Metadata.Authors {
			if author.Email != "" {
				parts = append(parts, fmt.Sprintf("%s <%s>", author.Name, author.Email))
			} else {
				parts = append(parts, author.Name)
			}
		}
		ann[ocispec.AnnotationAuthors] = strings.Join(parts, ", ")
	}
	if isSPDXExpression(pkg.License()) {
		ann[ocispec.AnnotationLicenses] = pkg.License()
	}
	if pkg.Card.Metadata.Vendor != "" {
		ann[ocispec.AnnotationVendor] = pkg.Card.Metadata.Vendor
	}
	if pkg.Card.Metadata.URL != "" {
		ann[ocispec.AnnotationURL] = pkg.Card.Metadata.URL
	}
	if pkg.Card.Metadata.Documentation != "" {
		ann[ocispec.AnnotationDocumentation] = pkg.Card.Metadata.Documentation
	}
	if len(pkg.Card.Metadata.Tags) > 0 {
		if data, err := json.Marshal(pkg.Card.Metadata.Tags); err == nil {
			ann[AnnotationTags] = string(data)
		}
	}
	if pkg.Compatibility() != "" {
		ann[AnnotationCompatibility] = pkg.Compatibility()
	}
	if pkg.AllowedTools() != "" {
		ann[AnnotationAllowedTools] = pkg.AllowedTools()
	}
	if pkg.Card.Metadata.Support != "" {
		ann[AnnotationSupport] = pkg.Card.Metadata.Support
	}
	if pkg.Card.Metadata.Changelog != "" {
		ann[AnnotationChangelog] = pkg.Card.Metadata.Changelog
	}
	if wordCount > 0 {
		ann[AnnotationWordCount] = strconv.Itoa(wordCount)
	}
	for key, value := range pkg.Card.Metadata.Annotations {
		ann[key] = value
	}
	return ann
}

func truncateAnnotation(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for len(value) > 0 && !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

// isSPDXExpression validates against the SPDX license list before mapping the
// free-form Agent Skills license value to the OCI SPDX-only annotation.
func isSPDXExpression(value string) bool {
	if value == "" {
		return false
	}
	valid, _ := spdxexp.ValidateLicenses([]string{value})
	return valid
}
