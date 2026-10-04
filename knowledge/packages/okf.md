---
type: Go Package
title: okf
description: Reads concept documents, records verifications and change requests, and checks a bundle.
resource: ../../internal/okf
tags: [bundle]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: okf
    resource: ../../internal/okf/okf.go
    title: internal/okf/okf.go
  - id: check
    resource: ../../internal/okf/check.go
    title: internal/okf/check.go
---

# API

| Identifier | Purpose |
|---|---|
| `Parse(text)`, `Load(root)` | one concept, or every concept under a bundle |
| `(Concept).NeedsReview()`, `HumanVerified()` | whether a person verified it since it was generated |
| `Verify`, `Unverify` | add or remove one actor's verification, leaving every other line as it was |
| `LoadRequests`, `SaveRequests`, `AddRequest`, `ResolveRequest`, `WithdrawRequest` | the change requests in `requests.json` |
| `Check(root)` | everything wrong with a bundle, one line per problem |
| `ValidActor(s)` | the actor convention |

# What Check checks

* every concept has `type`, `title`, `description` and `generated`, and
  every actor follows the convention;
* `status`, if present, is `draft`, `stable` or `deprecated`;
* every `resource`, source and `overrides` that is a path exists;
* every local markdown link resolves;
* every footnote matches a `sources` id, and every source is cited;
* every concept and folder is listed in its folder's `index.md` with the
  concept's own title and description;
* `log.md` uses ISO dates, newest first;
* `requests.json` is well-formed.[^check]

# Limits

Frontmatter is read by a small line parser, not a YAML library, to stay
free of dependencies. It understands `key: value` lines, the `generated`
and `verified` mappings, and the `sources` list. Other blocks are kept but
not interpreted.[^okf]

[^okf]: internal/okf/okf.go
[^check]: internal/okf/check.go
