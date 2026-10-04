---
type: Playbook
title: Review the knowledge
description: How a person checks the concepts agents wrote, verifies them or asks for changes.
tags: [knowledge, review]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:42:07Z }
sources:
  - id: kit
    resource: https://github.com/salkz/okf-kit
    title: okf-kit
---

# Why

A concept without a `verified` entry by a person is only an agent's reading
of the code. Verifying it tells later readers, people and agents, that
someone checked it.

# Start the tool

From the repository root, with your own id:

```
REVIEWER=<your id> docker compose -f docker-compose.review.yml up
```

Then open `http://<the machine's address>:8765`. With Go installed,
`./okf serve -reviewer <your id>` does the same without Docker.

# Steps

1. Choose **Review the next concept**.
2. Read the concept. The Sources links open the code it was derived from.
3. If it is right, press **Verify**.
4. If it is not, describe what should change under **Change requests** and
   press **Request a change**. The concept then waits for an agent.
5. Ask an agent to process the change requests; see
   [process change requests](process-change-requests.md).
6. Reload. Answered concepts show **updated, review again** with the
   agent's answer. Verify them, or request another change.
7. Stop the tool and commit the changed files in the bundle, including
   `requests.json`, or ask an agent to.

# Baseline concepts

Concepts under `baseline/` are marked **baseline** and are not part of the
review here. They are verified and changed where the kit is maintained.[^kit]

# Safety

The tool has no login. Anyone who can reach its port can verify as you and
leave requests, so run it only on a network you trust and stop it when you
are done.

# For agents

Do not verify concepts or submit change requests through the tool, and
never write a `human:` entry. Starting it for a person who asked is fine.

[^kit]: okf-kit
