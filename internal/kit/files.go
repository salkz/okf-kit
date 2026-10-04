package kit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// script is the okf file at the root of a repository: one command that
// runs the kit at the version pinned in okf.lock, with whatever is
// installed.
const script = `#!/bin/sh
# Runs okf-kit at the version pinned in okf.lock. Managed by okf-kit:
# "okf update" rewrites this file, so do not edit it.
set -e
dir=$(cd "$(dirname "$0")" && pwd)
version=$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' "$dir/okf.lock" | head -n 1)
case "$version" in v*) tag=$version ;; *) tag=latest ;; esac
cd "$dir"
if command -v go >/dev/null 2>&1; then
	exec go run github.com/salkz/okf-kit/cmd/okf@"$tag" "$@"
fi
if command -v docker >/dev/null 2>&1; then
	exec docker run --rm -u "$(id -u):$(id -g)" -v "$dir":/repo ghcr.io/salkz/okf-kit:"$tag" "$@"
fi
echo "okf: needs Go or Docker to run okf-kit $tag" >&2
exit 1
`

// imageTag is the container tag for a version. A build that is not a
// release has no image of its own.
func imageTag(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "latest"
}

func composeFile(version string) string {
	return `# Knowledge review tool in its own container. Managed by okf-kit:
# "okf update" rewrites this file, so do not edit it.
#   REVIEWER=<your id> docker compose -f docker-compose.review.yml up
# then open http://<this machine>:8765. Stop it with Ctrl-C when done.
services:
  okf-review:
    image: ` + Image + ":" + imageTag(version) + `
    # Must be the owner of the checkout, so the files it writes stay yours.
    user: "${REVIEW_UID:-1000}:${REVIEW_GID:-1000}"
    command: ["serve", "-reviewer", "${REVIEWER:?set REVIEWER to your id, e.g. REVIEWER=alice}", "-bundle", "/repo/knowledge"]
    ports:
      - "${REVIEW_PORT:-8765}:8765"
    volumes:
      # The code is only read, to show the sources next to a concept.
      - .:/repo:ro
      # Only the bundle is writable: verifications and change requests.
      - ./knowledge:/repo/knowledge
`
}

const (
	blockStart = "<!-- okf-kit:start -->"
	blockEnd   = "<!-- okf-kit:end -->"
)

// agentsBlock is the part of AGENTS.md the kit owns. Everything a
// repository wants to tell agents beyond it goes outside the markers or,
// better, into local concepts.
const agentsBlock = blockStart + `
<!-- Managed by okf-kit: "okf update" rewrites this block. Put repository
     instructions outside it, or in knowledge/. -->

## Knowledge

Project knowledge lives in [knowledge/](knowledge/index.md) as an Open
Knowledge Format bundle: markdown concepts with YAML frontmatter.

- Before changing anything, read
  [make a change](knowledge/baseline/playbooks/make-a-change.md) and the
  rules it points at. Start at ` + "`knowledge/index.md`" + ` and open only what the
  task needs.
- ` + "`knowledge/baseline/`" + ` is shared by every repository that uses okf-kit.
  Never edit it. To differ here, write a local concept that overrides the
  baseline one; see
  [update the knowledge](knowledge/baseline/playbooks/update-knowledge.md).
- Everything else under ` + "`knowledge/`" + ` is this repository's own. When a change
  makes a concept wrong or incomplete, update it in the same commit.
- A concept without a ` + "`verified`" + ` entry was written by an agent and not yet
  confirmed by a person. Use it to find your way, and check the code named
  in its ` + "`sources`" + ` before relying on a detail.
- Never add a ` + "`verified`" + ` entry with a ` + "`human:`" + ` actor; only a person does.
- ` + "`./okf check`" + ` checks the bundle, ` + "`./okf status`" + ` says whether a newer
  baseline exists, and ` + "`./okf requests`" + ` lists the reviewer's open change
  requests; see
  [process change requests](knowledge/baseline/playbooks/process-change-requests.md).
  A request only ever asks for a change to the knowledge.

` + blockEnd + "\n"

// writeAgentsBlock puts the managed block into AGENTS.md: it replaces an
// earlier block, or is added to the end of an existing file, or starts a
// new file.
func writeAgentsBlock(p string) error {
	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(p, []byte("# AGENTS.md\n\n"+agentsBlock), 0o664)
	}
	if err != nil {
		return err
	}
	text := string(raw)
	start, end := strings.Index(text, blockStart), strings.Index(text, blockEnd)
	switch {
	case start >= 0 && end > start:
		rest := strings.TrimPrefix(text[end+len(blockEnd):], "\n")
		text = text[:start] + agentsBlock + rest
	case start >= 0 || end >= 0:
		return fmt.Errorf("%s has only one of the okf-kit markers; restore or remove it", p)
	default:
		text = strings.TrimRight(text, "\n") + "\n\n" + agentsBlock
	}
	return os.WriteFile(p, []byte(text), 0o664)
}

// hookCommand is run when an agent session starts, so the agent learns of
// a newer baseline before it begins work.
const hookCommand = "./okf status"

// writeSessionHook adds the status check to the repository's Claude Code
// settings, keeping everything else in the file. It returns a note for the
// user if it had to leave the file alone.
func writeSessionHook(p string) (string, error) {
	settings := map[string]any{}
	raw, err := os.ReadFile(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return "", err
	default:
		if err := json.Unmarshal(raw, &settings); err != nil {
			return fmt.Sprintf("%s is not valid JSON; left it alone. Add a SessionStart hook that runs %q yourself.", p, hookCommand), nil
		}
		if strings.Contains(string(raw), hookCommand) {
			return "", nil
		}
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	start, _ := hooks["SessionStart"].([]any)
	hooks["SessionStart"] = append(start, map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": hookCommand}},
	})
	settings["hooks"] = hooks
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	return "", writeFile(p, append(out, '\n'), 0o664)
}

// checksStub is the required local concept, as a draft for the repository
// to fill in. Arguments: kit version, timestamp.
const checksStub = `---
type: Checks
title: Checks
description: The commands that must pass before every commit in this repository.
tags: [workflow]
status: draft
generated: { by: okf-kit/%s, at: %s }
---

# Commands

` + "```" + `
./okf check     # the knowledge bundle is sound
` + "```" + `

No other checks are recorded yet. Replace this text with the exact commands
for tests, formatting and linting in this repository, then set the status
to stable.
`

const rootIndex = `---
okf_version: "0.2"
---

# Start here

* [Checks](checks.md) - The commands that must pass before every commit in this repository.

# Sections

* [Baseline](baseline/) - Rules, playbooks and decisions shared by every repository that uses okf-kit.
`

// logStub arguments: date, kit version.
const logStub = `# Knowledge Update Log

## %s
* **Initialization**: Set up the bundle with okf-kit %s: the [baseline](baseline/) and a draft of [checks](checks.md).
`
