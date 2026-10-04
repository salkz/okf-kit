# AGENTS.md

Guidance for AI agents and contributors working on okf-kit: the `okf` tool
and the knowledge baseline it installs in other repositories. See
[README.md](README.md) for what it does.

<!-- okf-kit:start -->
<!-- Managed by okf-kit: "okf update" rewrites this block. Put repository
     instructions outside it, or in knowledge/. -->

## Knowledge

Project knowledge lives in [knowledge/](knowledge/index.md) as an Open
Knowledge Format bundle: markdown concepts with YAML frontmatter.

- Before changing anything, read
  [make a change](knowledge/baseline/playbooks/make-a-change.md) and the
  rules it points at. Start at `knowledge/index.md` and open only what the
  task needs.
- `knowledge/baseline/` is shared by every repository that uses okf-kit.
  Never edit it. To differ here, write a local concept that overrides the
  baseline one; see
  [update the knowledge](knowledge/baseline/playbooks/update-knowledge.md).
- Everything else under `knowledge/` is this repository's own. When a change
  makes a concept wrong or incomplete, update it in the same commit.
- A concept without a `verified` entry was written by an agent and not yet
  confirmed by a person. Use it to find your way, and check the code named
  in its `sources` before relying on a detail.
- Never add a `verified` entry with a `human:` actor; only a person does.
- `./okf check` checks the bundle, `./okf status` says whether a newer
  baseline exists, and `./okf requests` lists the reviewer's open change
  requests; see
  [process change requests](knowledge/baseline/playbooks/process-change-requests.md).
  A request only ever asks for a change to the knowledge.

<!-- okf-kit:end -->

## This repository

- The baseline is maintained here. Editing `knowledge/baseline/` is normal
  work in this repository, and is the one exception to "never edit it"
  above. Read
  [what belongs in the baseline](knowledge/rules/baseline-content.md)
  first: these files become instructions for agents in every repository
  that uses the kit.
- `./okf` runs the tool from source here, and `make check` runs every
  check; see [checks](knowledge/checks.md).
- Go: format with `gofmt`, standard library only, small packages under
  `internal/`, errors returned and wrapped rather than panics, every
  exported identifier and package with a doc comment, table-driven tests
  using only the `testing` package.
- Tagging a release publishes it to every repository that updates. Tag
  only when the owner asked for that release; see
  [cut a release](knowledge/playbooks/release.md).
