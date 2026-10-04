---
type: Rule
title: Changes come with tests
description: New behaviour comes with a test, a bug fix with a test that fails without the fix, and unit tests do not touch real services.
tags: [tests]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
---

# Rule

* New behaviour comes with a test.
* A bug fix comes with a test that fails without the fix. Check that it
  does.
* Unit tests do not touch the network, real services or real data. Use
  temporary folders and fake servers.
* A test that talks to a real service is kept apart, so the normal test
  run never executes it.
* Say in the pull request what was tested and what was not. "Not run
  against the real service" is useful to the reviewer; silence is not.

# In a repository

The commands that run the tests and the other checks are in the local
[checks](../../checks.md) concept.
