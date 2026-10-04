---
type: Go Package
title: cmd/okf
description: "The okf command: picks the subcommand, knows the version, and joins the two checks."
resource: ../../cmd/okf
tags: [cli]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: main
    resource: ../../cmd/okf/main.go
    title: cmd/okf/main.go
---

# Commands

| Command | Does | Implemented in |
|---|---|---|
| `init` | set a repository up | [kit](kit.md) |
| `update` | move it to this version | [kit](kit.md) |
| `status` | say whether a newer release exists | [kit](kit.md) |
| `check` | bundle check plus baseline integrity; exit 1 on a problem | here, [okf](okf.md) and [kit](kit.md) |
| `serve` | review site | [review](review.md) |
| `requests`, `resolve` | list and answer change requests | [review](review.md) |
| `version` | print the version | here |

Every command works on the current directory, which must be the repository
root.[^main]

# Version

The release build sets `version` with a linker flag. Run with
`go run <module>/cmd/okf@v1.2.3`, the version comes from the module's
build information instead. Anything else is `dev`.

# check

`check -upstream` is for this repository: the baseline may differ from any
lock because it is edited here. Draft concepts are listed as notes and do
not fail the check.

# Baseline in the review site

`serve` makes the baseline read-only exactly when the repository has an
`okf.lock`.

[^main]: cmd/okf/main.go
