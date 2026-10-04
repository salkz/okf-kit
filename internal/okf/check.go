package okf

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	actor     = regexp.MustCompile(`^(human:[\w.-]+|process:[\w.-]+|[\w.-]+/[\w.-]+)$`)
	mdLink    = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	footRef   = regexp.MustCompile(`\[\^([\w-]+)\]`)
	footDef   = regexp.MustCompile(`(?m)^\[\^([\w-]+)\]:`)
	indexItem = regexp.MustCompile(`^\* \[(.+)\]\(([^)]+)\) - (.+)$`)
	logDate   = regexp.MustCompile(`^## (.+)$`)
	codeFence = regexp.MustCompile("(?s)```.*?```")
	codeSpan  = regexp.MustCompile("`[^`\n]*`")
	rootIndex = regexp.MustCompile(`^okf_version: "\d+\.\d+"$`)
)

// ValidActor reports whether s follows the actor convention:
// <tool>/<version>, human:<id> or process:<id>.
func ValidActor(s string) bool {
	return actor.MatchString(s)
}

// checker collects the problems found in one bundle.
type checker struct {
	root     string
	problems []string
}

func (c *checker) errorf(format string, args ...any) {
	c.problems = append(c.problems, fmt.Sprintf(format, args...))
}

// rel names a file the way a reader of the report expects: relative to the
// bundle root.
func (c *checker) rel(path string) string {
	if r, err := filepath.Rel(c.root, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}

// Check reports everything wrong with the bundle at root, one line per
// problem, and nil if it is sound. It checks what a renderer or an agent
// relies on: required frontmatter, resolvable links and resource paths,
// footnotes that match sources, complete indexes, a dated log and a
// well-formed change requests file.
func Check(root string) ([]string, error) {
	c := &checker{root: root}
	concepts := map[string]Concept{}
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			dirs = append(dirs, path)
		case d.Name() == "log.md":
			c.checkLog(path)
		case strings.HasSuffix(path, ".md") && d.Name() != "index.md":
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			concept, err := Parse(string(raw))
			if err != nil {
				c.errorf("%s: %v", c.rel(path), err)
				return nil
			}
			concepts[path] = concept
			c.checkConcept(path, concept)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(concepts) == 0 {
		c.errorf("no concepts under %s", root)
	}
	for _, dir := range dirs {
		c.checkIndex(dir, concepts)
	}
	c.checkRequests(concepts)
	return c.problems, nil
}

// resolve turns a link or path found in the file at path into a filesystem
// path. Bundle-relative targets start with "/". ok is false for URLs.
func (c *checker) resolve(path, target string) (string, bool) {
	if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
		return "", false
	}
	target, _, _ = strings.Cut(target, "#")
	if target == "" {
		return path, true
	}
	if strings.HasPrefix(target, "/") {
		return filepath.Join(c.root, target), true
	}
	return filepath.Join(filepath.Dir(path), target), true
}

// prose removes code, where brackets and parentheses are not links.
func prose(body string) string {
	return codeSpan.ReplaceAllString(codeFence.ReplaceAllString(body, ""), "")
}

func (c *checker) checkLinks(path, body string) {
	for _, m := range mdLink.FindAllStringSubmatch(prose(body), -1) {
		target, local := c.resolve(path, m[1])
		if !local {
			continue
		}
		if _, err := os.Stat(target); err != nil {
			c.errorf("%s: link to %s does not resolve", c.rel(path), m[1])
		}
	}
}

func (c *checker) checkConcept(path string, concept Concept) {
	name := c.rel(path)
	for _, key := range []string{"type", "title", "description"} {
		if concept.Fields[key] == "" {
			c.errorf("%s: frontmatter has no %s", name, key)
		}
	}
	switch concept.Fields["status"] {
	case "", "draft", "stable", "deprecated":
	default:
		c.errorf("%s: status %q is not draft, stable or deprecated", name, concept.Fields["status"])
	}
	if concept.Generated.By == "" {
		c.errorf("%s: no generated entry", name)
	}
	for _, ev := range append([]Event{concept.Generated}, concept.Verified...) {
		if ev.By != "" && !ValidActor(ev.By) {
			c.errorf("%s: actor %q is not <tool>/<version>, human:<id> or process:<id>", name, ev.By)
		}
	}
	refs := concept.Resources()
	if o := concept.Fields["overrides"]; o != "" {
		refs = append(refs, o)
	}
	for _, r := range refs {
		if target, local := c.resolve(path, r); local {
			if _, err := os.Stat(target); err != nil {
				c.errorf("%s: path %s does not exist", name, r)
			}
		}
	}

	ids := map[string]bool{}
	for _, src := range concept.Sources {
		ids[src.ID] = true
	}
	text := prose(concept.Body)
	cited := map[string]bool{}
	// A definition line also looks like a reference, so it is taken out
	// before looking for the places a source is actually cited.
	for _, m := range footRef.FindAllStringSubmatch(footDef.ReplaceAllString(text, ""), -1) {
		cited[m[1]] = true
		if !ids[m[1]] {
			c.errorf("%s: footnote [^%s] has no sources entry with that id", name, m[1])
		}
	}
	defined := map[string]bool{}
	for _, m := range footDef.FindAllStringSubmatch(text, -1) {
		defined[m[1]] = true
	}
	for _, src := range concept.Sources {
		if !cited[src.ID] || !defined[src.ID] {
			c.errorf("%s: source %q needs a footnote reference and a footnote definition", name, src.ID)
		}
	}
	c.checkLinks(path, concept.Body)
}

// checkIndex makes sure an index.md lists exactly what its directory holds,
// with each concept's own title and description.
func (c *checker) checkIndex(dir string, concepts map[string]Concept) {
	path := filepath.Join(dir, "index.md")
	name := c.rel(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		c.errorf("%s: every directory needs an index.md", name)
		return
	}
	body := string(raw)
	if front, rest, ok := Split(body); ok {
		body = rest
		if dir != c.root || !rootIndex.MatchString(front) {
			c.errorf("%s: only the root index may have frontmatter, and only okf_version", name)
		}
	}
	c.checkLinks(path, body)

	listed := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		m := indexItem.FindStringSubmatch(line)
		if m == nil {
			if strings.HasPrefix(line, "*") {
				c.errorf("%s: entry %q is not \"* [Title](target) - description\"", name, line)
			}
			continue
		}
		title, target := m[1], m[2]
		listed[strings.TrimSuffix(target, "/")] = true
		concept, isConcept := concepts[filepath.Join(dir, target)]
		if !isConcept {
			continue
		}
		if title != concept.Fields["title"] || m[3] != concept.Fields["description"] {
			c.errorf("%s: entry for %s must repeat the concept's title and description", name, target)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		c.errorf("%s: %v", name, err)
		return
	}
	for _, e := range entries {
		n := e.Name()
		if n == "index.md" || n == "log.md" || (!e.IsDir() && !strings.HasSuffix(n, ".md")) {
			continue
		}
		if !listed[n] {
			c.errorf("%s: %s is not listed", name, n)
		}
	}
}

// checkLog makes sure log.md date headings are ISO dates, newest first.
func (c *checker) checkLog(path string) {
	name := c.rel(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		c.errorf("%s: %v", name, err)
		return
	}
	// Links in the log are not checked: it is history, and an entry may
	// name a concept that was removed since.
	var dates []string
	for _, line := range strings.Split(string(raw), "\n") {
		m := logDate.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if _, err := time.Parse("2006-01-02", m[1]); err != nil {
			c.errorf("%s: heading %q is not a YYYY-MM-DD date", name, m[1])
		}
		dates = append(dates, m[1])
	}
	if len(dates) == 0 {
		c.errorf("%s: no dated entries", name)
	}
	if !sort.SliceIsSorted(dates, func(i, j int) bool { return dates[i] > dates[j] }) {
		c.errorf("%s: dates must be newest first", name)
	}
}

// checkRequests makes sure the change requests file, which agents edit, is
// well-formed: unique ids, a known status, and for a done request a record
// of who answered it and how.
func (c *checker) checkRequests(concepts map[string]Concept) {
	reqs, err := LoadRequests(c.root)
	if err != nil {
		c.errorf("%v", err)
		return
	}
	seen := map[string]bool{}
	for _, r := range reqs {
		name := RequestsFile + ": request " + r.ID
		if r.ID == "" || seen[r.ID] {
			c.errorf("%s: id is empty or used twice", name)
		}
		seen[r.ID] = true
		if strings.TrimSpace(r.Text) == "" || !ValidActor(r.By) || r.At.IsZero() {
			c.errorf("%s: needs text, an actor in by, and a time in at", name)
		}
		switch r.Status {
		case RequestOpen:
			if _, ok := concepts[filepath.Join(c.root, filepath.FromSlash(r.Concept))]; r.Concept != "" && !ok {
				c.errorf("%s: concept %q does not exist", name, r.Concept)
			}
		case RequestDone:
			if strings.TrimSpace(r.Response) == "" || !ValidActor(r.ResolvedBy) || r.ResolvedAt == nil {
				c.errorf("%s: a done request needs response, resolvedBy and resolvedAt", name)
			}
		default:
			c.errorf("%s: status %q is not open or done", name, r.Status)
		}
	}
}
