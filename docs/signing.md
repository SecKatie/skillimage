# Sign and attest skill images with Cosign

SkillImage does not wrap Cosign. Build and push the image with `skillctl`, then
use the standard Cosign CLI so signatures and attestations stay interoperable
with the Sigstore ecosystem.

## Sign an immutable reference

Prefer signing a numbered prerelease, final version, or digest rather than a
mutable alias such as `latest`.

```bash
skillctl push ghcr.io/example/skills/pdf-processing:1.2.0-beta.2
cosign sign ghcr.io/example/skills/pdf-processing:1.2.0-beta.2
cosign verify ghcr.io/example/skills/pdf-processing:1.2.0-beta.2
```

Cosign supports keyless signing by default. Follow your organization's Sigstore
policy when a key, certificate identity, Rekor setting, or private Sigstore
deployment is required.

## Attach an in-toto attestation

Generate provenance with your build system, then attach it as an in-toto
attestation whose subject is the image digest:

```bash
cosign attest \
  --predicate provenance.json \
  --type slsaprovenance \
  ghcr.io/example/skills/pdf-processing@sha256:IMAGE_DIGEST

cosign verify-attestation \
  --type slsaprovenance \
  ghcr.io/example/skills/pdf-processing@sha256:IMAGE_DIGEST
```

The packaged `skill.yaml` intentionally has no provenance field. Signed
attestations provide verifiable provenance without inventing a second embedded
provenance format.
