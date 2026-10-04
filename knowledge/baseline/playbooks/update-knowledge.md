---
type: Playbook
title: Update the knowledge
description: "How to keep the bundle true: what to edit, when, how trust is recorded, and where local concepts go."
tags: [knowledge, agents]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
sources:
  - id: spec
    resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
    title: Open Knowledge Format specification
---

# When

In the same commit as the change that makes a concept wrong or incomplete.
Moving or renaming a file also means fixing the `resource` and `sources`
paths that name it; `./okf check` fails otherwise.

# Steps

1. Edit the concept. Keep it about what is true now; history goes in
   `log.md`.
2. Set `generated` to yourself and the current UTC time, for example
   `generated: { by: claude-code/<model>, at: 2026-10-04T12:00:00Z }`.
3. If you changed the title or description, copy them to the entry in the
   directory's `index.md`.
4. Add a line to the bundle's `log.md` under today's date, newest date
   first.
5. Run `./okf check`.

# Adding a concept

Create the file in the folder that fits, with at least `type`, `title`,
`description` and `generated`, and list it in that folder's `index.md`. Use
relative links. Name the code a concept was derived from in `sources`, and
cite it with a footnote whose label is the source's `id`.[^spec]

# Where things go

| Folder | Holds |
|---|---|
| `baseline/` | concepts that come with the kit; never edit them here |
| anything else | this repository's own concepts |

Common local folders are `packages/` or `components/`, `services/`,
`data/`, `flows/`, `rules/`, `playbooks/` and `decisions/`. Use what fits
the project.

# Differing from the baseline

Do not edit `baseline/`. To extend a baseline concept, write a local one
that links to it. To replace one in this repository, write a local concept
with `overrides: <path to the baseline concept>` in its frontmatter and
say why in its body. The local concept wins. See
[baseline and local knowledge](../decisions/baseline-and-local.md).

# Trust

* `generated.by` is `<tool>/<model>` for an agent and `human:<id>` for a
  person.
* Only a person adds a `verified` entry with a `human:` actor, normally
  with the [review tool](review-knowledge.md). An agent never writes one.
* A concept generated again after it was verified shows up for review
  again; do not remove the old `verified` entry yourself.

# What does not belong in a concept

Secrets, real hostnames or paths from someone's installation, and prose
that only repeats the code line by line.

[^spec]: Open Knowledge Format specification
