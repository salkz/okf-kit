---
type: Checks
title: Checks
description: The commands that must pass before every commit in this repository.
tags: [workflow]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: makefile
    resource: ../Makefile
    title: Makefile
---

# Commands

```
make check
```

It runs, in order:[^makefile]

| Command | Must |
|---|---|
| `gofmt -l .` | print nothing |
| `go vet ./...` | be clean |
| `go test ./...` | pass |
| `./okf check` | find this repository's bundle sound |

# Not covered locally

The container image and the release are built only by the GitHub
workflows. A change to the `Dockerfile` or to `.github/workflows/` is
first exercised when it is pushed.

[^makefile]: Makefile
