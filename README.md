# okf-kit

Keeps a project's knowledge in the repository, written by agents and
verified by a person, in the
[Open Knowledge Format](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md):
markdown concepts with YAML frontmatter.

It is two things released together:

- **The `okf` tool**: sets a repository up, checks the bundle, serves it for
  review in a browser, and handles the reviewer's change requests.
- **The baseline**: rules, playbooks and decisions that hold in every
  repository. Each repository gets a read-only copy and adds its own
  concepts beside it.

## Set up a repository

In the repository root, with Docker:

```
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD":/repo ghcr.io/salkz/okf-kit init
```

or with Go:

```
go run github.com/salkz/okf-kit/cmd/okf@latest init
```

This writes:

| File | What it is |
| --- | --- |
| `knowledge/baseline/` | the baseline; never edited in the repository |
| `knowledge/index.md`, `log.md`, `checks.md` | the start of the repository's own bundle |
| `okf.lock` | the kit version and a hash of every baseline file |
| `okf` | script that runs the pinned version with Go or Docker |
| `docker-compose.review.yml` | the review tool |
| `AGENTS.md` | a managed block that points agents at the bundle |
| `.claude/settings.json` | a session hook that runs `./okf status` |

Existing files keep their content: the block is added to an existing
`AGENTS.md`, the hook is merged into existing settings, and existing
concepts are left alone.

Then ask an agent to follow
`knowledge/baseline/playbooks/bootstrap-knowledge.md`.

## Day to day

```
./okf check       # is the bundle sound? run it with the other checks
./okf status      # is there a newer baseline?
./okf requests    # the reviewer's open change requests
```

Review in a browser:

```
REVIEWER=<your id> docker compose -f docker-compose.review.yml up
```

and open `http://<the machine>:8765`. Verify what is right; where a concept
is wrong, describe the change and ask an agent to process the change
requests. The tool has no login, so run it only on a network you trust.

## Updating

When `./okf status` reports a newer release, run that version's `update`
on its own branch:

```
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD":/repo ghcr.io/salkz/okf-kit:<version> update
```

It replaces the managed files and leaves a change request on every local
concept that overrides a baseline concept that changed. Review the diff in
a pull request: the baseline is instructions agents follow.

## Baseline and local concepts

- Never edit `knowledge/baseline/` in a repository; `./okf check` fails if
  it differs from `okf.lock`.
- To extend a baseline concept, write a local one that links to it.
- To replace one, write a local concept with
  `overrides: <path to the baseline concept>` in its frontmatter.

## This repository

The baseline is maintained here, in `knowledge/baseline/`. See
[knowledge/](knowledge/index.md) for how the tool is built and
[AGENTS.md](AGENTS.md) for working on it.

```
make check                                      # format, vet, tests, bundle check
make review REVIEWER=<your id>                  # review this repository's knowledge, with Go
REVIEWER=<your id> docker compose up --build    # the same, with Docker only
```

Here the review tool is built from source by `docker-compose.yml`, and the
baseline is reviewed like every other concept.

Licensed under Apache-2.0.
