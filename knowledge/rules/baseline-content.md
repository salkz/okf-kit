---
type: Rule
title: What belongs in the baseline
description: A baseline concept must hold in every repository, point only at URLs, and change with a version that says how big the change is.
tags: [baseline]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: test
    resource: ../../internal/kit/kit_test.go
    title: internal/kit/kit_test.go
---

# Rule

* A baseline concept is true in every repository, whatever its language
  or purpose. Anything about one project is a local concept there.
* Its `resource` and `sources` are URLs, never paths. It may link to other
  baseline concepts and to the local concepts listed as required in
  `kit.json`, nothing else. A test enforces the first part.[^test]
* Where repositories will legitimately differ, say so in the concept and
  leave the detail to a local one.
* The baseline is instructions that agents in other repositories will
  follow without asking. Do not put anything in it that the owner has not
  agreed to.

# Versions

One version covers the tool and the baseline.

| Change | Version |
|---|---|
| a rule removed or reversed, a required local concept added, a command or file format broken | major |
| a new concept, a new command, a rule made stricter in a compatible way | minor |
| wording, fixes | patch |

`okf status` tells agents elsewhere to stop and ask the owner on a major
version, so the choice matters.

# Editing

Edit the files under `knowledge/baseline/` here; they are embedded into
the binary as they are. Keep each folder's `index.md` in step, and add a
line to `log.md`.

[^test]: internal/kit/kit_test.go
