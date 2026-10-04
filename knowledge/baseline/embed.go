// Package baseline holds the baseline concepts that the kit copies into
// every repository. The files in this folder are the source of truth; edit
// them here and nowhere else.
package baseline

import "embed"

// Files is the baseline as it is copied into a repository's bundle. Go
// files and anything starting with "." or "_" are not part of it.
//
//go:embed index.md kit.json rules playbooks decisions
var Files embed.FS
