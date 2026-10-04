---
type: Go Package
title: review
description: "The review site: read a concept next to its sources, verify it or request a change."
resource: ../../internal/review
tags: [review, http]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: server
    resource: ../../internal/review/server.go
    title: internal/review/server.go
  - id: render
    resource: ../../internal/review/render.go
    title: internal/review/render.go
---

# Pages

| Route | Shows or does |
|---|---|
| `GET /` | every concept by folder, progress, requests about the whole bundle |
| `GET /c/<path>` | one concept with metadata, sources, body and its change requests |
| `POST /verify/<path>`, `/unverify/<path>` | add or remove the reviewer's verification |
| `POST /request/<path>` | record a change request; no path means the whole bundle |
| `POST /withdraw/<id>` | remove an open request |
| `GET /src/<path>` | a source file or folder the bundle points at |

The bundle is read from disk on every request.[^server]

# Review states

| Badge | Meaning | Offers Verify |
|---|---|---|
| unverified | no person has verified it | yes |
| change requested | a request is open | no |
| updated, review again | a request was answered and nobody verified since | yes |
| changed since it was verified | generated again after the last verification | yes |
| verified | verified and unchanged since | no |
| baseline | belongs to the baseline in a repository that uses the kit | no |

Baseline concepts cannot be verified or have changes requested there; the
page points upstream. A local concept with `overrides` is linked from the
baseline concept it replaces, and the other way round.

# Rendering

`render.go` renders the markdown the bundles use: headings, paragraphs,
flat lists, tables, fenced code, inline code, links, bold and footnotes.
Everything is HTML-escaped. Nested lists, images and block quotes come out
as plain text.[^render]

# Safety

* No login: anyone who can reach the port acts as the configured reviewer.
* Form posts from other websites are refused by checking `Origin` and
  `Sec-Fetch-Site`, and pages allow no scripts.
* It writes only `verified` lines and `requests.json`.
* The source viewer serves only files a concept names as a resource or
  links to, and the files directly inside such a folder, so the rest of
  the working tree stays private.

[^server]: internal/review/server.go
[^render]: internal/review/render.go
