---
type: Rule
title: Secrets stay out of code, logs and output
description: Credentials come from the environment or an ignored file, and never appear in code, logs, test output, commits or this bundle.
tags: [safety, secrets]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
---

# Rule

Passwords, tokens, session cookies and private keys come from the
environment or from a file that git ignores. They never appear in code,
logs, test output, commit messages, pull requests or knowledge concepts.

# For agents

* Do not print a file that holds secrets, or the value of a secret
  variable. Checking whether one is set is enough.
* Do not force-add an ignored file.
* Use made-up values in examples and tests.
* Never name a file that holds a secret as a `resource` or link to it from
  a concept: the review tool shows the files a bundle points at.

# In a repository

Which files and variables hold secrets is specific to the repository. A
local concept lists them; it extends this rule and does not replace it.
