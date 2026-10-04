---
type: Playbook
title: Update the baseline
description: How to move a repository to a newer version of the kit and its baseline.
tags: [knowledge, agents, baseline]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
sources:
  - id: kit
    resource: https://github.com/salkz/okf-kit
    title: okf-kit
---

# When

`./okf status` reports a newer release than the one pinned in `okf.lock`.

# Steps

1. Read what changed: `./okf status` prints the release notes.
2. If the new version is a major version, stop and tell the owner. It
   removes or reverses a rule, and that is their decision.
3. Branch, as in [branches](../rules/branches.md). Do not mix the update
   into other work.
4. Run the new version's update, for example
   `go run github.com/salkz/okf-kit/cmd/okf@<version> update` or
   `docker run --rm -u "$(id -u):$(id -g)" -v "$PWD":/repo ghcr.io/salkz/okf-kit:<version> update`.
5. It replaces `baseline/`, the managed block in `AGENTS.md`, the `okf`
   script, the review compose file and `okf.lock`.
6. For every local concept that overrides a baseline concept that changed,
   the update leaves a change request. Process them as in
   [process change requests](process-change-requests.md): decide whether
   the override still makes sense against the new baseline text.
7. Run `./okf check`, commit, and open a pull request. The owner merges.

# Rules

* Never edit files under `baseline/` or the hashes in `okf.lock` by hand.
  `./okf check` compares them and fails on a difference.
* The baseline is a set of instructions for agents, so an update changes
  what agents do. It comes only from the kit's own releases, and it is
  never merged without the owner seeing the diff.[^kit]

[^kit]: okf-kit
