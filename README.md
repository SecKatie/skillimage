# OCI Skill Registry

A framework-agnostic, OCI-based registry for AI agent skills.
Packages skills as standard OCI images, manages a SemVer prerelease lifecycle
(`alpha.N`, `beta.N`, `rc.N`, final), and works with
the container tools enterprises already use: `podman`, `skopeo`,
`crane`, and Kubernetes ImageVolumes.

Companion to
[agentoperations/agent-registry](https://github.com/agentoperations/agent-registry)
(metadata and governance layer).

## Why OCI images?

Skills are packaged as standard OCI images (`FROM scratch`), not
ORAS artifacts. This means:

- `podman pull` / `skopeo copy` / `crane pull` work out of the box
- Kubernetes ImageVolumes can mount skills directly into agent pods
- Standard registries (Quay, GHCR, Zot) index and serve them normally
- Signing and verification via standard Cosign/Sigstore workflows

## Install

### Homebrew (macOS / Linux)

```bash
brew install pavelanni/tap/skillctl
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/redhat-et/skillimage/main/install.sh | sh
```

To install a specific version or to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/redhat-et/skillimage/main/install.sh \
  | VERSION=0.1.0 INSTALL_DIR=~/.local/bin sh
```

### Go install

```bash
go install github.com/redhat-et/skillimage/cmd/skillctl@latest
```

### Container image

```bash
podman run --rm ghcr.io/redhat-et/skillctl:latest version
```

### From source

```bash
make build    # produces bin/skillctl
```

### GitHub releases

Pre-built binaries for Linux, macOS, and Windows (amd64/arm64)
are available on the
[releases page](https://github.com/redhat-et/skillimage/releases).

## Quick start

### Create a skill

A skill is an Agent Skills directory with an authoritative `SKILL.md` plus a
small `skill.yaml` containing distribution metadata. Initialize a minimal pair,
or include functional example resources with `--full`:

```bash
skillctl init ./hello-world
skillctl init ./hello-world --full
```

`SKILL.md` starts with Agent Skills frontmatter:

```markdown
---
name: hello-world
description: Greets users warmly. Use when a user asks for a greeting.
license: Apache-2.0
---

Greet the user warmly and ask how you can help.
```

The required v1alpha2 SkillCard is intentionally minimal:

```yaml
apiVersion: skillimage.io/v1alpha2
kind: SkillCard
metadata:
  version: 1.0.0
  title: Hello World
  vendor: Example Corp
  tags: [example, getting-started]
```

The Agent Skills fields `name`, `description`, `license`, `compatibility`,
`metadata`, and `allowed-tools` live only in `SKILL.md`. Repository location is
chosen through OCI references; it is not embedded as a namespace in the card.

### Validate, build, and inspect

```bash
# Validate both SKILL.md and skill.yaml
bin/skillctl validate examples/hello-world/

# First default build creates 1.0.0-alpha.1 and latest
bin/skillctl build examples/hello-world/

# Choose a remote-shaped repository and keep the managed lifecycle tags
bin/skillctl build -t ghcr.io/myorg/skills/hello-world examples/hello-world/

# List local images
bin/skillctl list

# Inspect metadata and OCI details
bin/skillctl inspect localhost/hello-world:latest

# Or create exactly one replaceable local reference
bin/skillctl build -t ghcr.io/myorg/skills/hello-world:canary examples/hello-world/
```

### Lifecycle promotion

```bash
# Promote local alpha -> beta (allocates beta.1 automatically)
bin/skillctl promote ghcr.io/myorg/skills/hello-world

# Fixes in beta remain beta and automatically advance beta.2, beta.3, ...
bin/skillctl build -t ghcr.io/myorg/skills/hello-world examples/hello-world/

# Promote to final (creates the unqualified base-version tag)
bin/skillctl promote ghcr.io/myorg/skills/hello-world --to final

# Explicitly move back to a lower stage when needed
bin/skillctl demote ghcr.io/myorg/skills/hello-world --to beta
```

Promotion updates OCI manifest annotations and retags without
modifying image content. The layer digest stays the same from
alpha through final. Local tags are replaceable workspace state. Conflicting
remote version tags are rejected unless `--force` is supplied.

### Push and pull (remote registry)

```bash
# Push the local effective version and latest to a remote registry
bin/skillctl push ghcr.io/myorg/skills/hello-world

# Pull remote latest and its effective version into the local store
bin/skillctl pull ghcr.io/myorg/skills/hello-world

# Build or promote locally, then publish in one command
bin/skillctl build -t ghcr.io/myorg/skills/hello-world examples/hello-world/ --push
bin/skillctl promote ghcr.io/myorg/skills/hello-world --push
```

An untagged repository uses managed behavior. An explicitly tagged reference
pushes or pulls exactly that tag. Push allows monotonic forward publication and
refuses remote-ahead or digest-conflict cases with pull and `--force` recovery
instructions.

Authentication uses your existing `~/.docker/config.json` or
Podman's `auth.json` -- no separate login needed.

### Install skills for AI agents

Install skills directly into an agent's skill directory. If the
image isn't in the local store, skillctl pulls it automatically.

```bash
# Install to Claude Code's skill directory
skillctl install quay.io/myorg/hello-world:1.0.0 --target claude

# Install to another agent
skillctl install quay.io/myorg/hello-world:1.0.0 --target opencode

# Install to a custom directory
skillctl install quay.io/myorg/hello-world:1.0.0 -o ~/custom/skills/
```

Supported targets: `claude`, `cursor`, `windsurf`, `opencode`,
`openclaw`.

For v1alpha2, skillctl records source, digest, and effective version outside the
immutable package at `<skills-root>/.skillimage/receipts/<skill-name>.json`.
Legacy v1alpha1 installations retain their embedded provenance behavior.

### List and upgrade installed skills

```bash
# List installed skills across all agent directories
skillctl list --installed

# List installed skills for a specific agent
skillctl list --installed --target claude

# Check which skills have newer published versions
skillctl list --installed --upgradable

# Upgrade a specific skill
skillctl upgrade hello-world --target claude

# Upgrade all skills for an agent
skillctl upgrade --all --target claude
```

### Remove local images

```bash
# Remove a skill image from the local store
skillctl rm examples/hello-world:1.0.0-draft

# Remove multiple images
skillctl rm ref1:tag ref2:tag

# Skip confirmation prompt
skillctl rm examples/hello-world:1.0.0-draft --force
```

### Inspect with standard tools

Skill images are standard OCI images. You can inspect metadata
from any registry without downloading the image:

```bash
# View all metadata annotations (no image download)
skopeo inspect docker://quay.io/myorg/hello-world:1.0.0

# Get just the skill tags
skopeo inspect docker://quay.io/myorg/hello-world:1.0.0 \
  | jq -r '.Annotations["io.skillimage.tags"]'
# → ["example","getting-started"]

# Get lifecycle status
skopeo inspect docker://quay.io/myorg/hello-world:1.0.0 \
  | jq -r '.Annotations["io.skillimage.status"]'
# → final
```

This works because all skill metadata is stored in OCI manifest
annotations, not inside the image layers. A catalog UI or CI
pipeline can read skill metadata with a single manifest fetch.

### Using skill images

Since skills are standard OCI images, you can mount them directly
into running containers with `--mount type=image`:

```bash
podman pull quay.io/myorg/hello-world:1.0.0
podman run --rm \
  --mount type=image,source=quay.io/myorg/hello-world:1.0.0,destination=/skills \
  my-agent:latest
```

This works with both Podman and Docker on Linux. On macOS, the
remote client does not support `--mount type=image`, but you
can create an image-backed volume instead:

```bash
podman pull quay.io/myorg/hello-world:1.0.0
podman volume create --driver image \
  --opt image=quay.io/myorg/hello-world:1.0.0 hello-world-skill
podman run --rm -v hello-world-skill:/skills:ro my-agent:latest
```

For Kubernetes / OpenShift, use
[ImageVolumes](https://kubernetes.io/docs/tasks/configure-pod-container/image-volumes/)
(beta in K8s 1.33) to mount skills directly into pods:

```yaml
volumes:
  - name: skill
    image:
      reference: quay.io/myorg/hello-world:1.0.0
      pullPolicy: IfNotPresent
containers:
  - name: agent
    volumeMounts:
      - name: skill
        mountPath: /skills
        readOnly: true
```

#### Example: mounting a skill into an agent runtime

This example pulls a document-summarizer skill from the registry
and mounts it into [OpenCode](https://github.com/anomalyco/opencode),
an open-source AI coding agent with a terminal UI:

```bash
podman pull quay.io/skillimage/business/document-summarizer:1.0.0-testing
podman volume create --driver image \
  --opt image=quay.io/skillimage/business/document-summarizer:1.0.0-testing \
  summarizer-skill
podman run -it --rm \
  -v summarizer-skill:/root/.config/opencode/skills/document-summarizer \
  ghcr.io/anomalyco/opencode
```

OpenCode discovers the skill automatically. Running `/skills`
in the TUI shows the document-summarizer, and the agent uses it
when asked to summarize a document or web page.

### Running skillctl on OpenShift

You can run skillctl directly on an OpenShift cluster to inspect
images in the internal registry. First, create a secret with your
registry credentials:

```bash
oc create secret docker-registry skillctl-auth \
  --docker-server=image-registry.openshift-image-registry.svc:5000 \
  --docker-username=unused \
  --docker-password="$(oc whoami -t)"
```

Then run skillctl as a pod with the secret mounted:

```bash
oc run skillctl --rm -i --restart=Never \
  --image=ghcr.io/redhat-et/skillctl:latest \
  --overrides='{"spec":{"containers":[{"name":"skillctl","image":"ghcr.io/redhat-et/skillctl:latest","args":["inspect","--tls-verify=false","image-registry.openshift-image-registry.svc:5000/NAMESPACE/SKILL@sha256:DIGEST"],"volumeMounts":[{"name":"auth","mountPath":"/home/skillctl/.docker"}]}],"volumes":[{"name":"auth","secret":{"secretName":"skillctl-auth","items":[{"key":".dockerconfigjson","path":"config.json"}]}}]}}'
```

Use `--tls-verify=false` for the internal registry (self-signed
certificate). The `-i` flag ensures `--rm` cleans up the pod
after it completes. The token from `oc whoami -t` is typically
valid for 24 hours; recreate the secret when it expires.

## Testing

```bash
make test        # Run all tests
make lint        # Run golangci-lint
make fmt         # Format code
```

## Project structure

| Path | Description |
| ---- | ----------- |
| `cmd/skillctl/` | CLI entry point |
| `internal/cli/` | Cobra commands |
| `pkg/skillcard/` | SkillCard parse, validate, serialize |
| `pkg/oci/` | OCI image build/push/pull/inspect/promote |
| `pkg/installed/` | Installed skill discovery and upgrade checking |
| `pkg/lifecycle/` | State machine, tag rules |
| `pkg/source/` | Remote Git source resolution |
| `schemas/` | JSON Schema for SkillCard |
| `examples/` | Sample skills |
| `docs/` | Design specs and research |

## Architecture

Library-first: core logic lives in `pkg/` as importable Go
packages. The CLI is a thin consumer. Agent runtimes, CI/CD
pipelines, and other tools can import the library directly.

```text
consumers: skillctl CLI, agent runtimes, CI/CD
      |
  pkg/ (public Go API)
  +-- skillcard/   parse, validate, serialize
  +-- oci/         build, push, pull, inspect, promote
  +-- installed/   scan, upgrade checking
  +-- lifecycle/   state machine, tag rules
  +-- source/      remote Git source resolution
      |
  OCI registries (quay.io, ghcr.io, Zot)
```

## Lifecycle stages

```text
alpha.N --> beta.N --> rc.N --> final
```

| Stage | OCI tag | Example |
| ----- | ------- | ------- |
| alpha | `<ver>-alpha.N` | `1.0.0-alpha.1` |
| beta | `<ver>-beta.N` | `1.0.0-beta.2` |
| release candidate | `<ver>-rc.N` | `1.0.0-rc.1` |
| final | `<ver>` | `1.0.0` |

Status is stored in OCI manifest annotations
(`io.skillimage.status`), not inside the image. Image content
is unchanged across transitions. A default build, or a build with an untagged
`-t` repository target, creates the numbered lifecycle tag and moves `latest`.
An explicitly tagged `-t` build creates only the requested reference. v1alpha1
retains its legacy draft/testing/published lifecycle and emits a deprecation
warning; see the [migration guide](docs/migrations/v1alpha1-to-v1alpha2.md).

For supply-chain signing and in-toto/SLSA attestations, use the standard
[Cosign workflow](docs/signing.md).

## Similar projects

Several projects explore packaging AI agent skills as OCI
artifacts. We share the same vision and welcome collaboration.

| Project | Author | Approach |
| ------- | ------ | -------- |
| [Agent Skills OCI Artifacts Spec](https://github.com/ThomasVitale/agents-skills-oci-artifacts-spec) | Thomas Vitale | Specification for skills as ORAS artifacts with Arconia CLI |
| [skills-oci](https://github.com/salaboy/skills-oci) | Mauricio Salatino | CLI for skills as OCI artifacts with SLSA provenance and SBOMs |
| **skillctl** (this project) | Red Hat OCTO | Skills as OCI images for multi-user OpenShift/K8s with lifecycle management, read-only ImageVolume mounting, and standard container tooling |

## License

Apache-2.0
