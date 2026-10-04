---
type: Rule
title: Commits are atomic and conventional
description: One logical change per commit, written as a Conventional Commit, with tests passing at every commit.
tags: [git, workflow]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
sources:
  - id: cc
    resource: https://www.conventionalcommits.org/en/v1.0.0/
    title: Conventional Commits 1.0.0
---

# Rule

* One logical change per commit. Do not mix a refactoring with a change in
  behaviour.
* The message follows Conventional Commits:[^cc]

  `<type>(<optional scope>): <summary>`

  followed by an optional body that says what changed and why.
* Types: `feat`, `fix`, `refactor`, `test`, `docs`, `build`, `chore`, `perf`.
* The summary is imperative, lower case, has no trailing period and is at
  most 72 characters.
* A breaking change has `!` after the type and a `BREAKING CHANGE:` footer.
* The checks pass at every commit, not only at the end of a branch.

# Why

A history of small, named changes can be reviewed, reverted and searched
one change at a time, by people and by agents.

# In a repository

The scopes in use, and anything that counts as breaking there, belong in a
local concept.

[^cc]: Conventional Commits 1.0.0
