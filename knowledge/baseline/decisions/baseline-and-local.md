---
type: Decision
title: Baseline and local knowledge
description: A read-only baseline that comes with the kit is shared by every repository; each repository adds its own concepts beside it.
tags: [knowledge, baseline]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
sources:
  - id: kit
    resource: https://github.com/salkz/okf-kit
    title: okf-kit
---

# Decision

A bundle has two layers:

| Layer | Where | Written by | Changed how |
|---|---|---|---|
| Baseline | `baseline/` | the kit's maintainer | a new kit release, then `okf update` |
| Local | everything else | agents working in the repository | in the repository, with the code |

# Why a copy in the repository

The baseline is copied into each repository rather than fetched when
needed. Agents and people see plain files with no extra step, and a
baseline update is an ordinary diff in a pull request.

# How the layers stay apart

* `okf.lock` records the kit version and a hash of every baseline file.
  `./okf check` fails if a baseline file was edited, added or removed.
* A local concept can extend a baseline concept by linking to it, or
  replace it with `overrides: <path>` in its frontmatter. The local one
  wins in that repository.
* When an update changes a baseline concept that is overridden, the
  override gets a change request so someone looks at it again.
* Baseline concepts are verified where the kit is maintained. The review
  tool shows them as **baseline** and keeps them out of the local review.

# What the baseline expects from a repository

A few concepts only the repository can write. Today that is one:
[checks](../../checks.md), the commands that must pass before a commit.
`./okf check` fails while a required concept is missing.[^kit]

[^kit]: okf-kit
