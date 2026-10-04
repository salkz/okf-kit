---
type: Project
title: okf-kit
description: Tool and shared baseline for keeping project knowledge as an Open Knowledge Format bundle that agents write and a person verifies.
resource: https://github.com/salkz/okf-kit
tags: [architecture]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: readme
    resource: ../README.md
    title: README
---

# What it is

Two things released together under one version:

* the `okf` tool, one static binary and a container image;
* the [baseline](baseline/), a set of rules, playbooks and decisions that
  every repository using the kit gets a read-only copy of.

A repository runs `okf init` once. Agents then write that repository's own
concepts beside the baseline, a person reviews them in a browser, and
`okf update` brings in a newer baseline.[^readme]

# Shape

| Part | Concept |
|---|---|
| Command line | [cmd/okf](packages/cmd.md) |
| Concept files, change requests, the bundle check | [okf](packages/okf.md) |
| Review site | [review](packages/review.md) |
| init, update, integrity, release check | [kit](packages/kit.md) |
| What every repository receives | [baseline](baseline/) |

`cmd/okf` uses the three packages; `review` and `kit` use `okf`; nothing
else depends on anything else. There are no third-party modules.

# This repository is special

It is where the baseline is written, so it has no `okf.lock`, its `./okf`
script runs the tool from source, and its review tool treats baseline
concepts like any other. Everywhere else the baseline is read-only.

# Before changing anything

Follow the baseline's [make a change](baseline/playbooks/make-a-change.md).
For the baseline itself, also read
[what belongs in the baseline](rules/baseline-content.md).

[^readme]: README
