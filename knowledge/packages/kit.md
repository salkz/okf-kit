---
type: Go Package
title: kit
description: Sets a repository up with the baseline, updates it, holds it to okf.lock and checks for a newer release.
resource: ../../internal/kit
tags: [baseline, release]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: kit
    resource: ../../internal/kit/kit.go
    title: internal/kit/kit.go
  - id: files
    resource: ../../internal/kit/files.go
    title: internal/kit/files.go
  - id: status
    resource: ../../internal/kit/status.go
    title: internal/kit/status.go
---

# API

| Function | Does |
|---|---|
| `Init(repo, version, now, w)` | writes the managed files and, where missing, a root index, a log and a draft `checks.md` |
| `Update(repo, version, now, w)` | rewrites the managed files; leaves a change request on each override of a changed baseline concept |
| `Check(repo, version, upstream)` | baseline integrity and required local concepts |
| `Status(repo, w, cache, now, fetch)` | compares the lock with the newest release |

# Managed files

Rewritten by both `Init` and `Update`:[^files]

| File | Content |
|---|---|
| `knowledge/baseline/` | the baseline embedded in the binary |
| `okf.lock` | kit version and a SHA-256 of each baseline file |
| `okf` | script that runs the pinned version with Go or Docker |
| `docker-compose.review.yml` | the review tool, pinned to the image of that version |
| `AGENTS.md` | only the block between the `okf-kit` markers |
| `.claude/settings.json` | only a `SessionStart` hook running `./okf status`, merged in |

Nothing else in a repository is touched.

# Integrity

`Check` fails if a baseline file was edited, added or removed against the
lock. If the lock names the running version, its hashes must also equal
the embedded baseline, which catches a lock edited to hide a change.[^kit]

# Status

It asks GitHub for the newest release, at most once a day, and prints the
release notes when the lock is behind. A major version tells the agent to
stop and ask. A network failure is reported in one line and never returns
an error.[^status]

# Things to keep

* `sync` removes the whole baseline folder before writing it. It must
  never be pointed at anything but `knowledge/baseline`.
* Existing local files are never overwritten by `Init`.

[^kit]: internal/kit/kit.go
[^files]: internal/kit/files.go
[^status]: internal/kit/status.go
