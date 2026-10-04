---
type: Rule
title: Work on a branch, merge through a pull request
description: Agents commit and push on their own branch without asking; the default branch changes only through a pull request the owner merges.
tags: [git, workflow, agents]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
---

# Rule

* Never commit to the default branch or push to it. It is protected.
* Work on a branch named for the change: `feat/<topic>`, `fix/<topic>` or
  `docs/<topic>`, started from an up-to-date default branch.
* An agent commits on its branch and pushes it without asking.
* An agent opens a pull request when the work is ready, also without
  asking. The description says what changed, what to look at, and what was
  and was not tested.
* The owner reviews and merges. An agent does not merge its own pull
  request.

# Why

Committing and pushing a branch changes nothing anyone depends on, so it
does not need a question. Merging does, so it stays with a person.

# Stacked work

When a change builds on a pull request that is not merged yet, branch from
that branch and open the new pull request against it, and say so in the
description.
