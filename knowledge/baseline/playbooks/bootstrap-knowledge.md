---
type: Playbook
title: Write the first local knowledge
description: How an agent fills an empty local bundle in a repository that already has code.
tags: [knowledge, agents]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
---

# When

`./okf init` was just run in a repository that has code but no local
concepts yet.

# Steps

1. Read the code before writing anything: the entry points, each package
   or component, the tests, the build and deployment files, and any
   existing contributor or agent instructions.
2. Write the [checks](../../checks.md) concept first: the exact commands
   that must pass before a commit.
3. Write an `overview.md`: what the project does and how the parts fit, in
   a page, linking to the rest.
4. Write one concept per package or component: what it provides, how it
   behaves, what must be kept intact, and what is known to be missing.
5. Write concepts for the outside systems it talks to, its data formats
   and settings, and its main flows step by step.
6. Write the repository's own rules, each with its reason and where the
   code upholds it. Where one differs from a baseline rule, use
   `overrides`, as in [update the knowledge](update-knowledge.md).
7. Record decisions you can establish from the history or from the owner,
   with their reasoning.
8. Run `./okf check`, then open a pull request.

# What good looks like

* Every concept names its `sources` and cites them.
* Gaps and risks found while reading are written down, not fixed in
  passing and not left out.
* Nothing is marked verified. The owner reviews the batch with the
  [review tool](review-knowledge.md).

# In a new repository

There is nothing to bootstrap. Concepts are written as the code is, by
[make a change](make-a-change.md).
