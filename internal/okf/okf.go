// Package okf reads and updates Open Knowledge Format concept documents:
// markdown files with a YAML frontmatter block, as kept in knowledge/.
//
// It reads frontmatter with a small line parser rather than a YAML library,
// so it only understands the simple forms the bundle uses.
package okf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Event records who generated or verified a concept, and when.
type Event struct {
	By string
	At time.Time
}

// Human reports whether the event was made by a person.
func (e Event) Human() bool {
	return strings.HasPrefix(e.By, "human:")
}

// Source is one entry of a concept's sources list.
type Source struct {
	ID, Resource, Title string
}

// Concept is one parsed concept document.
type Concept struct {
	// Fields holds the top-level frontmatter keys that have a value on
	// their own line, unquoted.
	Fields    map[string]string
	Generated Event // zero if the concept has no generated entry
	Verified  []Event
	Sources   []Source
	Body      string
}

var (
	topKey    = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$`)
	nestedKey = regexp.MustCompile(`^\s+(- )?([a-z_]+):\s*(.+)$`)
	eventMap  = regexp.MustCompile(`^\{\s*by:\s*(\S+?),\s*at:\s*(\S+?)\s*\}$`)
	listEvent = regexp.MustCompile(`^\s+- (\{.*\})\s*$`)
)

// timeLayout is how OKF timestamps are written: ISO 8601 in UTC.
const timeLayout = "2006-01-02T15:04:05Z"

// unquote strips the double quotes YAML needs around values containing ": ".
func unquote(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(s[1 : len(s)-1])
	}
	return s
}

// Split separates the frontmatter block of a document from its body. ok is
// false if text does not start with a closed frontmatter block.
func Split(text string) (front, body string, ok bool) {
	rest, found := strings.CutPrefix(text, "---\n")
	if !found {
		return "", text, false
	}
	front, body, ok = strings.Cut(rest, "\n---\n")
	return front, body, ok
}

func parseEvent(s string) (Event, error) {
	m := eventMap.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Event{}, fmt.Errorf("%q is not { by: <actor>, at: <time> }", s)
	}
	at, err := time.Parse(time.RFC3339, m[2])
	if err != nil {
		return Event{}, fmt.Errorf("%q is not an ISO 8601 time with an offset", m[2])
	}
	return Event{By: m[1], At: at}, nil
}

// Parse reads a concept document.
func Parse(text string) (Concept, error) {
	front, body, ok := Split(text)
	if !ok {
		return Concept{}, errors.New("no frontmatter block")
	}
	c := Concept{Fields: map[string]string{}, Body: body}
	block := "" // the top-level key whose indented lines are being read
	for _, line := range strings.Split(front, "\n") {
		if m := topKey.FindStringSubmatch(line); m != nil {
			key, val := m[1], m[2]
			block = ""
			switch {
			case val == "":
				block = key
			case key == "generated":
				ev, err := parseEvent(val)
				if err != nil {
					return Concept{}, fmt.Errorf("generated: %w", err)
				}
				c.Generated = ev
			case key == "verified":
				// A bare mapping is a one-element list.
				ev, err := parseEvent(val)
				if err != nil {
					return Concept{}, fmt.Errorf("verified: %w", err)
				}
				c.Verified = append(c.Verified, ev)
			default:
				c.Fields[key] = unquote(val)
			}
			continue
		}
		if block == "" || !strings.HasPrefix(line, " ") {
			return Concept{}, fmt.Errorf("frontmatter line %q is not a form this parser reads", line)
		}
		switch block {
		case "verified":
			m := listEvent.FindStringSubmatch(line)
			if m == nil {
				return Concept{}, fmt.Errorf("verified entry %q is not \"- { by, at }\"", line)
			}
			ev, err := parseEvent(m[1])
			if err != nil {
				return Concept{}, fmt.Errorf("verified: %w", err)
			}
			c.Verified = append(c.Verified, ev)
		case "sources":
			m := nestedKey.FindStringSubmatch(line)
			if m == nil {
				return Concept{}, fmt.Errorf("sources line %q is not \"key: value\"", line)
			}
			if m[1] != "" {
				c.Sources = append(c.Sources, Source{})
			}
			if len(c.Sources) == 0 {
				return Concept{}, fmt.Errorf("sources line %q comes before the first \"- \" entry", line)
			}
			src := &c.Sources[len(c.Sources)-1]
			switch val := unquote(m[3]); m[2] {
			case "id":
				src.ID = val
			case "resource":
				src.Resource = val
			case "title":
				src.Title = val
			}
		}
	}
	return c, nil
}

// Resources returns every resource the concept names: its own and those of
// its sources.
func (c Concept) Resources() []string {
	var out []string
	if r := c.Fields["resource"]; r != "" {
		out = append(out, r)
	}
	for _, s := range c.Sources {
		if s.Resource != "" {
			out = append(out, s.Resource)
		}
	}
	return out
}

// HumanVerified returns the latest verification by a person.
func (c Concept) HumanVerified() (Event, bool) {
	var latest Event
	found := false
	for _, ev := range c.Verified {
		if ev.Human() && (!found || ev.At.After(latest.At)) {
			latest, found = ev, true
		}
	}
	return latest, found
}

// NeedsReview reports whether no person has verified the concept, or its
// content was generated again after the last time one did.
func (c Concept) NeedsReview() bool {
	ev, ok := c.HumanVerified()
	return !ok || c.Generated.At.After(ev.At)
}

// Verify returns the document text with a verification by actor at the
// given time. An earlier verification by the same actor is replaced.
func Verify(text, actor string, at time.Time) (string, error) {
	return setVerified(text, actor, &Event{By: actor, At: at})
}

// Unverify returns the document text without any verification by actor.
func Unverify(text, actor string) (string, error) {
	return setVerified(text, actor, nil)
}

// setVerified rewrites the verified block: it drops actor's entries, adds
// add if it is set, and leaves every other line of the document untouched.
func setVerified(text, actor string, add *Event) (string, error) {
	c, err := Parse(text)
	if err != nil {
		return "", err
	}
	var events []Event
	for _, ev := range c.Verified {
		if ev.By != actor {
			events = append(events, ev)
		}
	}
	if add != nil {
		events = append(events, *add)
	}

	front, body, _ := Split(text)
	var kept []string
	at := -1 // where the new block goes
	inBlock := false
	for _, line := range strings.Split(front, "\n") {
		switch {
		case strings.HasPrefix(line, "verified:"):
			inBlock = true
			at = len(kept)
		case inBlock && strings.HasPrefix(line, " "):
		default:
			inBlock = false
			kept = append(kept, line)
			if at < 0 && strings.HasPrefix(line, "generated:") {
				at = len(kept)
			}
		}
	}
	if at < 0 {
		at = len(kept)
	}

	var block []string
	format := func(ev Event) string {
		return fmt.Sprintf("{ by: %s, at: %s }", ev.By, ev.At.UTC().Format(timeLayout))
	}
	switch len(events) {
	case 0:
	case 1:
		block = []string{"verified: " + format(events[0])}
	default:
		block = []string{"verified:"}
		for _, ev := range events {
			block = append(block, "  - "+format(ev))
		}
	}
	lines := append(append(append([]string{}, kept[:at]...), block...), kept[at:]...)
	return "---\n" + strings.Join(lines, "\n") + "\n---\n" + body, nil
}

// Doc is a concept together with where it was found.
type Doc struct {
	// Path is the concept's path inside the bundle, with forward slashes,
	// for example "packages/app.md".
	Path string
	Concept
}

// Load parses every concept under the bundle directory root, sorted by
// path with the concepts at the root first. index.md and log.md are not
// concepts. A file that does not parse is an error.
func Load(root string) ([]Doc, error) {
	var docs []Doc
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() || !strings.HasSuffix(name, ".md") || name == "index.md" || name == "log.md" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		c, err := Parse(string(raw))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		docs = append(docs, Doc{Path: filepath.ToSlash(rel), Concept: c})
		return nil
	})
	sort.SliceStable(docs, func(i, j int) bool {
		iRoot, jRoot := !strings.Contains(docs[i].Path, "/"), !strings.Contains(docs[j].Path, "/")
		if iRoot != jRoot {
			return iRoot
		}
		return docs[i].Path < docs[j].Path
	})
	return docs, err
}
