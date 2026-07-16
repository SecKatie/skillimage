# SkillCard v1alpha2 and Agent Skills Conformance

**Date:** 2026-07-15
**Status:** Approved design
**API version:** `skillimage.io/v1alpha2`

## Summary

SkillCard v1alpha2 makes `SKILL.md` the authoritative definition of an
Agent Skill and reduces `skill.yaml` to portable SkillImage packaging and
catalog metadata.

The release also adds strict Agent Skills validation, complete skill-directory
packaging, an initialization command, SemVer prerelease lifecycle tags, and a
clear separation between package metadata, OCI location, installation state,
and signed supply-chain attestations.

The design follows these principles:

- `SKILL.md` owns skill identity, activation metadata, and instructions.
- `skill.yaml` owns only SkillImage distribution and catalog metadata.
- OCI references determine registry and repository location.
- OCI digests provide immutable artifact identity.
- Local installation receipts record operational state.
- in-toto attestations signed with Cosign provide provenance evidence.
- The no-argument CLI path is opinionated and safe; explicit arguments give
  the user direct control without hidden aliases.

## Goals

- Support the complete Agent Skills package format, including `scripts/`,
  `references/`, `assets/`, dotfiles, and arbitrary additional content.
- Implement all current Agent Skills frontmatter fields and constraints.
- Make Agent Skills validation strict by default with an explicit compatibility
  escape hatch.
- Eliminate duplicated or dead SkillCard fields.
- Preserve read, validation, and build compatibility for v1alpha1 cards.
- Provide a straightforward `skillctl init` authoring experience.
- Replace lifecycle suffixes such as `-draft` and `-testing` with ordered SemVer
  prereleases.
- Follow Podman conventions for explicitly supplied build references.
- Avoid inventing a SkillImage-specific provenance format.

## Non-goals

- Automatically rewriting v1alpha1 cards.
- Providing a migration command.
- Embedding Cosign signing or verification inside `skillctl` in this change.
- Automatically promoting arbitrary Agent Skills metadata into OCI annotations.
- Treating a registry path as a Kubernetes-like namespace or vendor identity.
- Introducing SkillCard behavioral configuration to replace `SKILL.md`.

## External specifications

- Agent Skills specification: <https://agentskills.io/specification>
- OCI annotations: <https://github.com/opencontainers/image-spec/blob/main/annotations.md>
- Semantic Versioning 2.0.0: <https://semver.org/spec/v2.0.0.html>
- Podman build CLI: <https://docs.podman.io/en/stable/markdown/podman-build.1.html>
- Cosign attestations: <https://github.com/sigstore/cosign/blob/main/doc/cosign_attest.md>

## Authoritative sources and boundaries

### `SKILL.md`

`SKILL.md` is required and authoritative for:

- `name`
- `description`
- `license`
- `compatibility`
- `metadata`
- `allowed-tools`
- Markdown instructions
- References to bundled scripts, documentation, assets, and other resources

The skill name comes only from `SKILL.md`. v1alpha2 does not include a second
`metadata.name` field.

### `skill.yaml`

`skill.yaml` is required and authoritative for:

- The target base version for the release line
- Optional human-readable catalog presentation
- Optional publisher and URL metadata
- Optional namespaced OCI annotations

It does not contain behavioral configuration, registry location, installation
state, or provenance claims.

### OCI reference and digest

The OCI reference supplied at build, tag, push, or pull time determines the
registry and repository location. The digest identifies the immutable artifact.
Registry paths are not embedded into v1alpha2 package metadata, allowing the
same artifact to be mirrored without rewriting its content.

### Local installation receipt

Source reference, resolved digest, installation time, target, and installed
version are local operational state. They are stored outside the installed
skill directory and never written back into packaged `skill.yaml`.

### Signed provenance

Build provenance belongs in a signed in-toto/SLSA attestation whose subject is
the OCI artifact digest. Documentation will show standard `cosign sign`,
`cosign attest`, and verification workflows. SkillImage will not place
self-asserted provenance inside v1alpha2 cards.

## v1alpha2 schema

Example:

```yaml
apiVersion: skillimage.io/v1alpha2
kind: SkillCard
metadata:
  version: 1.2.0
  title: PDF Processing
  vendor: Bumbleforge
  tags:
    - pdf
    - documents
  authors:
    - name: Example Org
      email: skills@example.com
  url: https://example.com/pdf-skill
  documentation: https://docs.example.com/pdf-skill
  support: https://github.com/example/pdf-skill/issues
  changelog: https://example.com/pdf-skill/changelog
  annotations:
    com.example.department: document-automation
```

### Required fields

| Field | Constraint |
| --- | --- |
| `apiVersion` | Exactly `skillimage.io/v1alpha2` |
| `kind` | Exactly `SkillCard` |
| `metadata.version` | Strict `MAJOR.MINOR.PATCH` SemVer without prerelease or build identifiers |

`metadata.version` is the target base version, such as `1.2.0`. The effective
artifact version is a generated SemVer value such as `1.2.0-beta.2` until final
publication.

### Optional catalog fields

| Field | Purpose |
| --- | --- |
| `metadata.title` | Human-readable title distinct from `SKILL.md.name` |
| `metadata.vendor` | Human-readable distributing organization |
| `metadata.tags` | Search and discovery keywords |
| `metadata.authors` | Responsible people or organizations |
| `metadata.url` | Project or skill homepage |
| `metadata.documentation` | Documentation URL |
| `metadata.support` | Support or issue URL |
| `metadata.changelog` | Release notes or changelog URL |
| `metadata.annotations` | Organization-specific string annotations |

URL fields must use absolute HTTPS URLs. HTTP is accepted only for localhost.

Annotation keys must use reverse-domain notation. User annotations may not use
the reserved `org.opencontainers.*` or `io.skillimage.*` namespaces and may not
collide with generated annotations.

### Removed v1alpha1 fields

The following fields do not exist in v1alpha2:

- `metadata.name`
- `metadata.namespace`
- `metadata.description`
- `metadata.license`
- `metadata.compatibility`
- `metadata.allowed-tools`
- `provenance`
- `spec.prompt`
- `spec.examples`
- `spec.dependencies`

`display-name` is renamed to `title`. Skill-defined values move to or remain in
the authoritative `SKILL.md`. Dead `spec` fields are removed rather than moved.

## Versioned parsing architecture

Persisted formats and runtime behavior remain separate:

```text
skill.yaml v1alpha1 -> alpha1 adapter --+
                                        |
skill.yaml v1alpha2 -> alpha2 adapter --+-> NormalizedPackage
SKILL.md ------------> Agent Skills ----+          |
                                                   +-> OCI packaging
                                                   +-> annotations/catalog
                                                   +-> installation
```

### v1alpha1 adapter

- Preserves the existing schema and semantics.
- Remains readable, valid, and buildable.
- Uses legacy naming, packaging, and lifecycle behavior.
- Emits one deprecation warning per command with a link to
  `docs/migrations/v1alpha1-to-v1alpha2.md`.

### v1alpha2 adapter

- Parses the minimal SkillCard envelope.
- Requires and parses `SKILL.md`.
- Enforces Agent Skills conformance by default.
- Produces the same internal normalized package used downstream.

### Agent Skills parser

The parser supports the complete current frontmatter contract:

- Required `name`, with length, character, hyphen, and directory-name rules
- Required `description`, including its length constraint
- Optional free-form or file-reference `license`
- Optional length-constrained `compatibility`
- Optional string-to-string `metadata`
- Optional space-separated `allowed-tools`

The Markdown body is preserved without format restrictions. Unknown fields are
strict-validation findings until supported by the parser.

### Normalized package

The normalized package is an internal model, not another serialized manifest.
It combines versioned card data with parsed Agent Skills data and supplies a
version-independent input to OCI packaging, annotation generation, cataloging,
and installation.

## Validation behavior

Strict validation is the default for v1alpha2.

The `--allow-nonconformant` flag is available on `init`, `validate`, and
`build`. It downgrades Agent Skills conformance failures to warnings, but does
not bypass:

- Missing `SKILL.md`
- Invalid or missing v1alpha2 fields
- Missing usable package identity
- Invalid base SemVer
- Unreadable packaged content
- Broken or cyclic symlinks
- Archive path traversal
- Invalid or reserved custom annotations

When a nonconformant `SKILL.md` has no usable valid name, the logical directory
name is the compatibility fallback. `validate --allow-nonconformant` exits zero
when all findings were downgraded Agent Skills findings.

Unknown SkillCard API versions fail with a supported-version list. Parsers never
silently treat an unknown version as the latest version.

## Initialization

```text
skillctl init [directory]
skillctl init [directory] --full
```

The default directory is `.`.

### Existing `SKILL.md`

`init` parses and validates the existing `SKILL.md`, then creates a minimal
v1alpha2 card. `--version` sets the base version and defaults to `0.1.0`.

```yaml
apiVersion: skillimage.io/v1alpha2
kind: SkillCard
metadata:
  version: 0.1.0
```

No values are inferred from arbitrary `SKILL.md.metadata`, Git remotes,
usernames, or environment state.

### Missing `SKILL.md`

The directory basename supplies the new skill name unless `--name` is given.
`init` creates a valid starter `SKILL.md` and matching v1alpha2 `skill.yaml`.

### Full example

`--full` additionally creates:

```text
scripts/example.sh
references/REFERENCE.md
assets/example-template.md
```

The shell script is executable and noninteractive, supports `--help`, and emits
a small deterministic result. The generated `SKILL.md` links to and explains all
three resources.

### Collision handling

Initialization is atomic with respect to managed paths:

1. Calculate all files that would be created or replaced.
2. If any target exists, write nothing.
3. Print the complete collision list.
4. Explain that `--force` can replace those files.
5. With `--force`, replace only known generated targets and leave unrelated
   files untouched.

## Alpha2 packaging

v1alpha2 packages the complete skill directory.

- Include all regular files and directories, including dotfiles.
- Preserve executable permission bits.
- Normalize archive ownership so host user and group data do not leak.
- Reject sockets, devices, named pipes, and other unsupported special files.
- Validate archive paths against traversal.

### Symlinks

Symlinks are intentional package inputs and may resolve anywhere readable,
including outside the skill directory or when the skill directory itself is
reached through a symlink.

- Fully resolve links before reading.
- Copy resolved files and directory trees under the logical archive path.
- Emit only ordinary files and directories, never symlink archive entries.
- Track filesystem identities to detect cycles.
- Reject broken links.
- Report every external target included in the package.
- Do not store the external source path in the artifact.

This behavior can package deliberately linked sensitive data. Visibility and
review are the control because external dereferencing is an explicit feature.

v1alpha1 retains its legacy packaging exclusions for compatibility.

## OCI annotation projection

### Standard annotations

| OCI annotation | Authoritative source |
| --- | --- |
| `org.opencontainers.image.title` | `metadata.title`, otherwise `SKILL.md.name` |
| `org.opencontainers.image.description` | `SKILL.md.description` |
| `org.opencontainers.image.version` | Effective generated artifact SemVer |
| `org.opencontainers.image.authors` | `metadata.authors` |
| `org.opencontainers.image.vendor` | `metadata.vendor`; omitted when absent |
| `org.opencontainers.image.url` | `metadata.url` |
| `org.opencontainers.image.documentation` | `metadata.documentation` |
| `org.opencontainers.image.created` | Generated at build time |

### License caveat

Agent Skills allows `license` to contain free-form text or a bundled license
file reference, while the OCI licenses annotation expects an SPDX license
expression.

- Keep `SKILL.md.license` authoritative under Agent Skills rules.
- Emit `org.opencontainers.image.licenses` only when the value parses as a
  valid SPDX license expression.
- Retain non-SPDX values in package content without rejecting the package.
- Emit an informational diagnostic when the OCI annotation is omitted.

### SkillImage annotations

| Annotation | Source |
| --- | --- |
| `io.skillimage.status` | Current lifecycle stage |
| `io.skillimage.tags` | `metadata.tags` |
| `io.skillimage.compatibility` | `SKILL.md.compatibility` |
| `io.skillimage.allowed-tools` | `SKILL.md.allowed-tools` |
| `io.skillimage.wordcount` | Computed from the Markdown body |
| `io.skillimage.support` | `metadata.support` |
| `io.skillimage.changelog` | `metadata.changelog` |
| `io.skillimage.skillcard-version` | `skillimage.io/v1alpha2` |

Validated user annotations are added without overriding generated annotations.

Agent Skills arbitrary metadata remains in `SKILL.md`. Registry synchronization
does not automatically turn it into annotations or download layers merely to
index it.

## Repository and reference behavior

v1alpha2 contains no embedded namespace or repository prefix. OCI references
select both the repository and the operation mode:

- An untagged repository is a managed lifecycle operation.
- An explicitly tagged reference is an exact OCI operation.
- Podman-style short names are normalized by prepending `localhost/` when no
  registry is present.
- Digest references are rejected as build targets because the build digest
  does not exist yet.

Examples:

```text
skillctl build -t pdf-processing .
-> localhost/pdf-processing:1.2.0-alpha.1
-> localhost/pdf-processing:latest

skillctl build -t bumbleforge.com/example/pdf-processing:canary .
-> bumbleforge.com/example/pdf-processing:canary
```

An explicitly tagged build creates exactly one local reference, may replace an
existing local tag, and does not move local `latest`. Local tags are workspace
state and are intentionally replaceable. An explicit push operates only on the
requested remote tag.

For a multi-skill Git source, `-t` remains invalid because a single repository
or exact reference cannot name multiple artifacts.

## SemVer lifecycle

### Stages

The canonical lifecycle is:

```text
alpha.N -> beta.N -> rc.N -> final
```

Examples:

```text
1.2.0-alpha.1
1.2.0-alpha.2
1.2.0-beta.1
1.2.0-beta.2
1.2.0-rc.1
1.2.0
```

Dot-separated numeric identifiers are used so SemVer compares prerelease
numbers numerically.

### Managed build path

Without `-t`, or with an untagged `-t` repository target, `skillctl build`
follows the managed local lifecycle path:

1. Use repository `localhost/<SKILL.md.name>` by default, or the normalized
   untagged `-t` repository when supplied.
2. Resolve that repository's local `latest` reference.
3. Compare its effective base version to `metadata.version`.
4. If no matching version exists, build `alpha.1`.
5. If the base version matches, remain in the current prerelease stage and
   allocate its next available number.
6. If the matching version is final, fail and require a base-version bump.
7. Create both the effective-version tag and local `latest`.

Examples:

```text
No existing 1.2.0       -> 1.2.0-alpha.1
latest is 1.2.0-alpha.3 -> 1.2.0-alpha.4
latest is 1.2.0-beta.2  -> 1.2.0-beta.3
latest is 1.2.0-rc.1    -> 1.2.0-rc.2
latest is 1.2.0         -> fail; bump metadata.version
```

Build never accesses a registry unless `--push` is supplied. `build --push`
keeps the successful local build even if publication fails.

### Overrides

Advanced and CI workflows may override automatic allocation:

```text
skillctl build . --stage beta
skillctl build . --stage beta --prerelease-number 3
```

`--stage` can intentionally skip the promotion ladder. An explicit number must
be positive and fails if its immutable tag already exists.

### Promotion

Promotion is local by default, uses the managed repository's local `latest`,
and advances one stage automatically:

```text
skillctl promote bumbleforge.com/example/pdf-processing
alpha.N -> beta.1 -> rc.1 -> final
```

`--to` remains available for an intentional forward skip. Cross-stage
promotion uses `.1` when the target stage is unused, otherwise the next
available target-stage number. Final promotion creates the unqualified base
version tag. Promoting an already-final base version fails and requires a base
version bump.

Promotion does not access the registry. `promote --push` promotes locally and
then publishes the result. Direct remote promotion is not part of the managed
workflow; pull first when the desired source exists only in the registry. The
existing `--local` flag is redundant under this model and may remain as a
deprecated no-op during the compatibility window.

### Demotion

`skillctl demote` is also a local lifecycle operation. It moves only to a lower
stage, retains all historical tags, and allocates the next available number in
the target stage.

```text
skillctl demote localhost/pdf-processing --to beta
-> localhost/pdf-processing:1.2.0-beta.2
```

Like promotion, demotion moves `latest` only when it currently points to the
source being transitioned. A subsequent argument-free build then remains in
the demoted stage and allocates its next number.

### Local and remote cursors

Local `latest` is the private working lifecycle cursor. Remote `latest` is the
most recently published cursor. Local state may move ahead through builds and
promotions without being shared:

```text
local latest:  1.2.0-rc.1
remote latest: 1.2.0-beta.2
```

This is valid. The next managed push may publish the local cursor because it is
a monotonic forward move. Catalog and upgrade logic must interpret remote
`latest` as the latest published cursor, while build and local lifecycle logic
use local `latest`. The lifecycle annotation is authoritative for maturity.

v1alpha1 retains its legacy draft/testing/published behavior, including its
published-to-latest behavior.

## Push behavior

An untagged push is managed:

```text
skillctl push bumbleforge.com/example/pdf-processing
```

It reads local `latest`, derives the effective version from the manifest
annotation, validates remote state, pushes the effective-version tag first,
and moves remote `latest` only after that succeeds. An explicitly tagged push
continues to push exactly one reference.

Managed push allows a monotonic forward publication. It refuses when remote
`latest` is ahead, when the local base version is older, or when the same
effective version has a different digest. A refusal reports both states and
recommends either `skillctl pull <repository>` or an intentional `--force`.

`--force` may replace both a conflicting remote version tag and remote
`latest`. The flag is sufficient authorization and does not prompt, making it
usable in CI. Output records the old and new digests. Without `--force`, a push
to an existing version tag with the same digest succeeds idempotently.

OCI registries do not provide a transaction across two tags. All conflict
checks complete before mutation. If the effective version succeeds but moving
`latest` fails, the version remains published and rerunning the push safely
completes the operation.

## Pull behavior

An untagged pull synchronizes the managed published cursor:

```text
skillctl pull bumbleforge.com/example/pdf-processing
```

It resolves remote `latest`, reads its effective version annotation, copies the
content once, and creates both the local effective-version tag and local
`latest`. An explicitly tagged pull continues to pull exactly one reference.

Pull is idempotent when state matches and accepts a remote cursor that is ahead
of local state. It refuses to discard unpublished local progress or accept the
same version with a different digest. `pull --force` intentionally moves local
`latest` to the remote cursor while retaining existing local version tags.

## Managed workflow

The intended private-to-published workflow is:

```text
skillctl build -t bumbleforge.com/example/pdf-processing .
skillctl build -t bumbleforge.com/example/pdf-processing .
skillctl promote bumbleforge.com/example/pdf-processing
skillctl build -t bumbleforge.com/example/pdf-processing .
skillctl promote bumbleforge.com/example/pdf-processing
skillctl push bumbleforge.com/example/pdf-processing
```

This may privately advance through `alpha.N`, `beta.N`, or `rc.N` before the
first publication. `--push` on build or promote publishes the resulting local
cursor using the same managed push behavior.

Successful managed commands list every affected reference. Conflict errors
include the local and remote versions, relevant digests, and exact pull or
force recovery commands. Publication failure never rolls back a successful
local build or promotion.

## Catalog identity

For v1alpha2, catalog identity and grouping come from observed OCI state:

- Registry
- Repository path
- Parsed `SKILL.md.name`
- Effective artifact version
- Digest

The catalog does not rely on a self-declared namespace. Vendor is an optional
display and filtering field and is never inferred from the repository path.

## Installation receipts

Receipts are stored under each skills root rather than inside an individual
skill:

```text
<skills-root>/.skillimage/receipts/<skill-name>.json
```

A receipt records:

- Source OCI reference
- Resolved digest
- Installation timestamp
- Agent or custom target
- Installed package version

Listing and upgrade checks use receipts. New installations can reconstruct
missing state from the resolved reference and digest. Existing v1alpha1
installations remain discoverable through the legacy adapter.

## Deprecation and migration

All v1alpha1 uses emit one warning per command:

```text
warning: skillimage.io/v1alpha1 is deprecated; see
docs/migrations/v1alpha1-to-v1alpha2.md
```

`skillctl init` always creates v1alpha2. There is no migration command.

The migration guide will explain how to:

- Ensure Agent Skills metadata exists in `SKILL.md`.
- Remove duplicated SkillCard fields.
- Rename `display-name` to `title` where desired.
- Rename `homepage` to `url` if present in intermediate drafts.
- Remove `spec` and `provenance`.
- Replace namespace-derived local references with explicit OCI tagging.
- Adopt the SemVer prerelease lifecycle.

## Security considerations

- Strict validation is the default.
- The nonconformance flag does not disable filesystem or archive safety.
- External symlink dereferencing is visible but intentionally not sandboxed to
  the skill root.
- Archive paths are sanitized during pack and unpack.
- Custom annotations cannot override reserved identity, lifecycle, or security
  data.
- Provenance claims are not trusted merely because they appear in package
  content.
- Consumers should verify Cosign signatures and attestations against explicit
  identity and issuer policy.
- Published numbered and final tags reject conflicting replacement by default.
  `--force` is an explicit escape hatch, so consumers should still pin digests
  for strongest guarantees.

## Extensibility

- New optional Agent Skills fields can be supported in the Agent Skills parser
  without changing the SkillCard API version.
- New optional SkillImage catalog fields may be added compatibly to alpha2 when
  their semantics are stable.
- Breaking SkillCard semantics require a future API version.
- Organization-specific discovery metadata belongs in validated custom
  annotations.
- Behavioral configuration remains in `SKILL.md` or bundled resources.

## Testing strategy

### Schema and compatibility

- Dispatch alpha1 and alpha2 to separate schemas.
- Reject unknown versions.
- Verify alpha1 behavior remains unchanged apart from one deprecation warning.

### Agent Skills validation

- Cover every standard frontmatter field and constraint.
- Cover name-to-directory matching and malformed YAML.
- Cover strict and nonconformant modes.
- Verify missing `SKILL.md` remains fatal.

### Initialization

- Generate minimal alpha2 from an existing `SKILL.md`.
- Generate a valid example pair when it is absent.
- Execute and verify the `--full` example resources.
- Verify collision detection writes nothing.
- Verify `--force` touches only managed targets.

### Packaging

- Include dotfiles.
- Dereference internal and external file and directory symlinks.
- Detect broken links and cycles.
- Reject special files and traversal.
- Preserve executable bits and normalize ownership.
- Verify external host paths do not appear in the archive.
- Verify pull produces no symlinks.

### Lifecycle and references

- Drive implementation with a command-level TDD workflow covering private
  alpha iteration, promotion to beta, beta fixes, promotion to rc, and final
  publication.
- Allocate alpha, beta, and rc numbers automatically from local `latest`.
- Remain in the inferred local stage.
- Cover automatic promotion, demotion, skipped stages, overrides, and final
  publication.
- Verify local `latest` is the working cursor and remote `latest` is the
  published cursor.
- Require a version bump after final publication.
- Verify no `-t`, and untagged `-t`, create the effective-version tag plus
  local `latest`.
- Verify explicitly tagged builds create exactly one replaceable local
  reference.
- Verify managed push publishes the effective version and `latest` in order.
- Cover remote-absent, matching, local-ahead, remote-ahead, rollback, and
  same-version digest-conflict cases.
- Verify `--force` replaces conflicting remote version and `latest` tags and
  reports old and new digests.
- Verify managed pull restores both the effective-version tag and local
  `latest` without discarding unpublished work by default.
- Verify partial publication is safely retryable.
- Verify build and promotion remain local when `--push` fails.
- Verify explicit push and pull references remain exact one-reference
  operations.
- Reject digest build targets.
- Cover reference parsing for registry ports, nested paths, explicit tags,
  digests, and omitted tags.

### Metadata and catalog

- Verify every typed OCI annotation mapping.
- Emit the OCI license annotation only for valid SPDX expressions.
- Preserve non-SPDX license values in package content.
- Never infer vendor.
- Reject reserved or colliding custom annotations.
- Derive alpha2 catalog grouping from observed repository state.

### Operational state and documentation

- Store receipts outside skill directories.
- Drive list and upgrade behavior from receipts.
- Keep alpha1 installations discoverable.
- Validate all migration and generated examples.
- Exercise documented Cosign workflows against a disposable local registry
  where CI permits.

## Completion criteria

The design is satisfied when:

- v1alpha2 cards contain only approved distribution and catalog metadata.
- Strictly conforming Agent Skills packages round-trip through build and pull
  with all resources intact.
- v1alpha1 remains usable with deprecation warnings.
- The argument-free workflow manages numbered SemVer prereleases and `latest`.
- Explicit tagging creates only the requested reference.
- Managed push and pull synchronize the effective-version tag and `latest`
  without losing unpublished local progress by default.
- Build and promotion remain local unless `--push` is supplied.
- Installation and provenance data no longer mutate packaged SkillCards.
- Documentation explains migration, Cosign attestations, lifecycle behavior,
  and manual-control escape hatches.
