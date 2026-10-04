---
type: Decision
title: Keep project knowledge as an OKF bundle
description: Knowledge an agent needs and cannot cheaply read off the code lives in the repository as an Open Knowledge Format bundle.
tags: [knowledge, agents]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
sources:
  - id: spec
    resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
    title: Open Knowledge Format specification
---

# Decision

Why a rule exists, how outside systems behave, what was decided and how
tasks are done is kept in the repository as an Open Knowledge Format
bundle: markdown concepts with YAML frontmatter, in which only `type` is
required.[^spec]

# Why this format

* It is plain files in the repository: readable without tooling, diffable,
  and changed in the same commit as the code.
* It records where a concept came from (`sources`), who wrote it
  (`generated`), who confirmed it (`verified`) and whether it is current
  (`status`). That matters when agents write most of the content.

# How it is used

* The agent instructions file stays a short list and points at the bundle.
* Every concept names the code it was derived from.
* A concept written by an agent is unverified until a person has read it
  against the code.
* `./okf check` checks structure, links and paths, and runs with the
  repository's other checks.

# Not used

Attested computations and `stale_after`. Concepts here go stale when code
changes, not on a date, so the path check and the
[update playbook](../playbooks/update-knowledge.md) cover it.

[^spec]: Open Knowledge Format specification
