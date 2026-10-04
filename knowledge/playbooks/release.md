---
type: Playbook
title: Cut a release
description: How a new version of the tool and baseline is published.
tags: [release]
status: stable
generated: { by: claude-code/claude-fable-5-1, at: 2026-10-04T17:47:14Z }
sources:
  - id: workflow
    resource: ../../.github/workflows/release.yml
    title: .github/workflows/release.yml
---

# Steps

1. Everything for the release is merged to `main` and CI is green.
2. Choose the version by the table in
   [what belongs in the baseline](../rules/baseline-content.md).
3. Tag that commit `v<major>.<minor>.<patch>` and push the tag.
4. The release workflow tests, builds the binaries, pushes the image as
   `ghcr.io/salkz/okf-kit:<tag>` and `:latest`, and creates the GitHub
   release with generated notes.[^workflow]
5. Edit the release notes so the first lines say what changed in the
   baseline: `okf status` prints them to agents in other repositories.

# Who

The owner decides that a release happens. An agent may push the tag when
the owner said so for that release.

# After the first release

The container package on GitHub starts out private. It has to be made
public once, in the package's settings, for other machines to pull it
without logging in.

[^workflow]: .github/workflows/release.yml
