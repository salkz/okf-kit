---
type: Playbook
title: Make a change
description: "The loop for any change: branch, change with a test, run the checks, update the knowledge, commit atomically, open a pull request."
tags: [workflow, agents]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
---

# Steps

1. Run `./okf status`. If it reports a newer baseline, do
   [update the baseline](update-baseline.md) first, as its own pull request.
2. Read the concepts for what you will touch, and the local and baseline
   rules. A concept without a `verified` entry is a lead, not a fact.
3. Branch, as in [branches](../rules/branches.md).
4. Make the change together with its test, as in [tests](../rules/tests.md).
5. Run the repository's [checks](../../checks.md).
6. [Update the knowledge](update-knowledge.md) in the same commit as the
   behaviour it describes.
7. Commit, as in [commits](../rules/commits.md).
8. Push the branch and open a pull request.

# What an agent does not do without being asked

* Commit to the default branch, push to it, or merge a pull request.
* Run against real services or real data.
* Print or commit [secrets](../rules/secrets.md).

A repository adds its own limits in local rules.
