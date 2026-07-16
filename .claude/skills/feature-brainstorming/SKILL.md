---
name: feature-brainstorming
description: >-
  Structured discovery-to-design workflow for turning a rough idea into an
  approved architecture before any implementation. Explores repo/PR context,
  asks one clarifying question at a time (prefer A/B/C/D), proposes 2–3
  approaches with a recommendation, then presents the design section by
  section for approval and writes a design doc/ADR only after sign-off.
  Use when designing features, rewriting ADRs, aligning with external
  patterns (e.g. go-plugin), brainstorming architecture, or when the user
  says "brainstorm", "design this", "explore approaches", or wants a
  design before coding. Do not implement until the written design is
  approved.
---

You are helping me design a feature through a structured discovery → brainstorm → design process. Do NOT use skills, plugins, or external workflow tools for this. Follow ONLY the process below.

## Goal

Turn my rough idea into a clear design I approve before any implementation. No code, scaffolding, commits, or implementation plans until I explicitly approve the written design.

## Process (strict order)

### 1. Explore context

- Inspect the relevant repo/docs/PRs/issues/comments.
- Summarize in a few sentences: current state, constraints, and what the idea seems to be asking for.
- Create a short checklist of the steps below and work through them in order.

### 2. Clarifying questions (one at a time)

- Ask exactly ONE question per message.
- Prefer multiple-choice (A/B/C/D) with a short gloss for each option.
- Focus on purpose, constraints, success criteria, scope, packaging, and non-goals.
- Do not propose approaches until clarifying questions are sufficiently answered.
- After each answer, briefly acknowledge it, then ask the next single question.
- Stop asking when you can honestly propose approaches without guessing major forks.
- Provide your recommended option for each question, but allow the user to choose.

### 3. Propose 2–3 approaches

- Present 2–3 concrete options with trade-offs.
- Lead with a recommendation and why.
- Ask me which approach (or hybrid) to take.
- Do not start the full design until I pick one.

### 4. Present the design section by section

Cover, as relevant:

- Architecture / host–plugin (or component) boundaries
- Discovery, naming, manifests, packaging
- Interfaces / RPCs / data flow for the primary user journey
- Error handling, versioning, security boundaries
- Extensibility / how new kinds or features plug in later
- Testing strategy

Rules:

- One section per message (or a short section + “does this look right so far?”).
- Wait for my approval or edits before the next section.
- If I challenge an assumption, revise that section before continuing.
- Keep YAGNI ruthless: defer what isn’t needed for v1, but say how it extends later.

### 5. Write the design artifact

Only after I approve the design sections:

- Write a design doc (and/or ADR if that’s what we decided) to the path we agreed.
- Do a quick self-review for placeholders, contradictions, ambiguity, and scope.
- Ask me to review the written file before any implementation planning.

### 6. Stop at design approval

- Do not implement, open PRs, or write an implementation plan unless I ask.

## Communication style

- Direct and concise.
- Bold sparingly.
- Prefer pointed options over essays.
- When exploring an existing PR/issue, ground recommendations in what that artifact already proposed and what should change.
