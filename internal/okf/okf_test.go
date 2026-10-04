package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantErr  bool
		title    string
		res      int
		verified int
	}{
		{"plain", "---\ntype: Rule\ntitle: A\n---\nbody", false, "A", 0, 0},
		{"quoted value", "---\ntype: Rule\ntitle: \"A: b\"\n---\n", false, "A: b", 0, 0},
		{"sources", "---\ntype: Rule\ntitle: A\nresource: ../x\nsources:\n  - id: s\n    resource: ../y\n    title: \"Y: z\"\n---\n", false, "A", 2, 0},
		{"verified mapping", "---\ntype: Rule\ntitle: A\nverified: { by: human:me, at: 2026-10-04T10:00:00Z }\n---\n", false, "A", 0, 1},
		{"verified list", "---\ntype: Rule\ntitle: A\nverified:\n  - { by: human:me, at: 2026-10-04T10:00:00Z }\n  - { by: process:nightly, at: 2026-10-05T10:00:00+03:00 }\n---\n", false, "A", 0, 2},
		{"unknown block is kept", "---\ntype: Rule\ntitle: A\nextra:\n  - anything\n---\n", false, "A", 0, 0},
		{"no frontmatter", "# Heading\n", true, "", 0, 0},
		{"unclosed", "---\ntype: Rule\n", true, "", 0, 0},
		{"stray line", "---\ntype: Rule\n- stray\n---\n", true, "", 0, 0},
		{"bad time", "---\ntype: Rule\ngenerated: { by: a/b, at: yesterday }\n---\n", true, "", 0, 0},
	}
	for _, tt := range tests {
		c, err := Parse(tt.text)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if err == nil && (c.Fields["title"] != tt.title || len(c.Resources()) != tt.res || len(c.Verified) != tt.verified) {
			t.Errorf("%s: got title %q, %d resources, %d verified", tt.name, c.Fields["title"], len(c.Resources()), len(c.Verified))
		}
	}
}

func TestNeedsReview(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	tests := []struct {
		name     string
		verified []Event
		want     bool
	}{
		{"never verified", nil, true},
		{"only a process", []Event{{By: "process:nightly", At: day(9)}}, true},
		{"person, after it was generated", []Event{{By: "human:me", At: day(6)}}, false},
		{"person, before it was generated again", []Event{{By: "human:me", At: day(4)}}, true},
		{"latest person counts", []Event{{By: "human:a", At: day(4)}, {By: "human:b", At: day(7)}}, false},
	}
	for _, tt := range tests {
		c := Concept{Generated: Event{By: "tool/1", At: day(5)}, Verified: tt.verified}
		if got := c.NeedsReview(); got != tt.want {
			t.Errorf("%s: NeedsReview = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestVerify(t *testing.T) {
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.FixedZone("EEST", 3*3600))
	const stamp = "{ by: human:me, at: 2026-10-04T09:00:00Z }"
	const other = "{ by: process:nightly, at: 2026-10-01T02:00:00Z }"
	head := "---\ntype: Rule\ntitle: \"A: b\"\ngenerated: { by: tool/1, at: 2026-10-03T00:00:00Z }\n"
	tail := "sources:\n  - id: s\n    resource: ../x\n---\n\n# Body\n\n---\nnot frontmatter\n"
	tests := []struct {
		name   string
		before string
		want   string
	}{
		{"first verification goes after generated", "", "verified: " + stamp + "\n"},
		{"same person again is replaced", "verified: { by: human:me, at: 2026-01-01T00:00:00Z }\n", "verified: " + stamp + "\n"},
		{"mapping by someone else becomes a list", "verified: " + other + "\n", "verified:\n  - " + other + "\n  - " + stamp + "\n"},
		{"list is extended", "verified:\n  - " + other + "\n  - { by: human:x, at: 2026-10-02T00:00:00Z }\n",
			"verified:\n  - " + other + "\n  - { by: human:x, at: 2026-10-02T00:00:00Z }\n  - " + stamp + "\n"},
	}
	for _, tt := range tests {
		got, err := Verify(head+tt.before+tail, "human:me", at)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if want := head + tt.want + tail; got != want {
			t.Errorf("%s: got\n%s\nwant\n%s", tt.name, got, want)
		}
		// Taking the verification away again restores what was there,
		// except that a replaced older entry of the same person is gone.
		back, err := Unverify(got, "human:me")
		if err != nil {
			t.Errorf("%s: Unverify: %v", tt.name, err)
			continue
		}
		c, err := Parse(back)
		if err != nil {
			t.Errorf("%s: after Unverify: %v", tt.name, err)
			continue
		}
		if _, ok := c.HumanVerified(); ok && !strings.Contains(tt.before, "human:x") {
			t.Errorf("%s: Unverify left a human verification:\n%s", tt.name, back)
		}
	}

	if _, err := Verify("# no frontmatter\n", "human:me", at); err == nil {
		t.Error("Verify accepted a document without frontmatter")
	}
}

// writeBundle creates a small sound bundle and returns its root.
func writeBundle(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	files := map[string]string{
		"src/code.go":        "package src\n",
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n# Start\n\n* [Overview](overview.md) - The overview.\n* [Rules](rules/) - Rules.\n",
		"knowledge/log.md":   "# Log\n\n## 2026-10-04\n* **Creation**: [Overview](overview.md).\n\n## 2026-10-01\n* Older.\n",
		"knowledge/overview.md": "---\ntype: Project\ntitle: Overview\ndescription: The overview.\nresource: ../src\n" +
			"generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\nsources:\n  - id: code\n    resource: ../src/code.go\n    title: code.go\n---\n\n" +
			"See the [rule](rules/safe.md) and the code.[^code]\n\n[^code]: code.go\n",
		"knowledge/rules/index.md": "# Rules\n\n* [Stay safe](safe.md) - A rule.\n",
		"knowledge/rules/safe.md": "---\ntype: Rule\ntitle: Stay safe\ndescription: A rule.\nstatus: stable\n" +
			"generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\n---\n\nBack to the [overview](/overview.md). `[not](a-link.md)`\n",
	}
	for rel, text := range files {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o775); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o664); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(repo, "knowledge")
}

func TestLoad(t *testing.T) {
	docs, err := Load(writeBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].Path != "overview.md" || docs[1].Path != "rules/safe.md" {
		t.Fatalf("Load returned %d docs, first %q; want the root concept first and no reserved files", len(docs), docs[0].Path)
	}
}

func TestCheck(t *testing.T) {
	if problems, err := Check(writeBundle(t)); err != nil || len(problems) != 0 {
		t.Fatalf("a sound bundle has problems: %v, %v", problems, err)
	}

	// Each case breaks the sound bundle in one way.
	edit := func(rel, old, new string) func(*testing.T, string) {
		return func(t *testing.T, root string) {
			t.Helper()
			p := filepath.Join(root, filepath.FromSlash(rel))
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), old) {
				t.Fatalf("%s does not contain %q", rel, old)
			}
			if err := os.WriteFile(p, []byte(strings.Replace(string(raw), old, new, 1)), 0o664); err != nil {
				t.Fatal(err)
			}
		}
	}
	write := func(rel, text string) func(*testing.T, string) {
		return func(t *testing.T, root string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(text), 0o664); err != nil {
				t.Fatal(err)
			}
		}
	}
	tests := []struct {
		name   string
		break_ func(*testing.T, string)
		want   string
	}{
		{"missing type", edit("rules/safe.md", "type: Rule\n", ""), "rules/safe.md: frontmatter has no type"},
		{"bad status", edit("rules/safe.md", "status: stable", "status: done"), "status \"done\""},
		{"no generated", edit("rules/safe.md", "generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\n", ""), "no generated entry"},
		{"bad actor", edit("rules/safe.md", "by: tool/1", "by: someone"), "actor \"someone\""},
		{"moved resource", edit("overview.md", "resource: ../src\n", "resource: ../gone\n"), "path ../gone does not exist"},
		{"moved source", edit("overview.md", "../src/code.go", "../src/gone.go"), "path ../src/gone.go does not exist"},
		{"missing override target", edit("rules/safe.md", "status: stable", "overrides: ../baseline/x.md"), "path ../baseline/x.md does not exist"},
		{"broken link", edit("overview.md", "(rules/safe.md)", "(rules/nope.md)"), "link to rules/nope.md does not resolve"},
		{"uncited source", edit("overview.md", "code.[^code]", "code."), "source \"code\" needs a footnote"},
		{"unknown footnote", edit("rules/safe.md", "Back to", "Back[^x] to"), "footnote [^x]"},
		{"unlisted concept", edit("rules/index.md", "* [Stay safe](safe.md) - A rule.\n", ""), "rules/index.md: safe.md is not listed"},
		{"index text differs", edit("rules/index.md", "- A rule.", "- Another."), "must repeat the concept's title and description"},
		{"frontmatter in a sub-index", edit("rules/index.md", "# Rules", "---\nokf_version: \"0.2\"\n---\n# Rules"), "only the root index may have frontmatter"},
		{"file without frontmatter", write("rules/stray.md", "# stray\n"), "rules/stray.md: no frontmatter block"},
		{"log out of order", edit("log.md", "## 2026-10-01", "## 2026-11-01"), "dates must be newest first"},
		{"log bad date", edit("log.md", "## 2026-10-01", "## yesterday"), "is not a YYYY-MM-DD date"},
		{"bad requests file", write(RequestsFile, "{not json"), RequestsFile},
		{"request for a missing concept", write(RequestsFile, `[{"id":"r1","concept":"gone.md","text":"x","by":"human:me","at":"2026-10-04T00:00:00Z","status":"open"}]`), "concept \"gone.md\" does not exist"},
		{"done request without a response", write(RequestsFile, `[{"id":"r1","text":"x","by":"human:me","at":"2026-10-04T00:00:00Z","status":"done"}]`), "a done request needs response"},
	}
	for _, tt := range tests {
		root := writeBundle(t)
		tt.break_(t, root)
		problems, err := Check(root)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !strings.Contains(strings.Join(problems, "\n"), tt.want) {
			t.Errorf("%s: problems %q do not mention %q", tt.name, problems, tt.want)
		}
	}
}

func TestRequests(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 4, 12, 0, 0, 500, time.FixedZone("EEST", 3*3600))

	reqs, err := LoadRequests(dir)
	if err != nil || len(reqs) != 0 {
		t.Fatalf("a bundle without a requests file gave %v, %v", reqs, err)
	}
	reqs, first := AddRequest(reqs, "packages/app.md", "wait polls every 5 s, not 3", "human:me", at)
	reqs, second := AddRequest(reqs, "", "add a concept about releases", "human:me", at)
	if first.ID != "r1" || second.ID != "r2" || !first.Open() || first.At.Location() != time.UTC {
		t.Errorf("AddRequest gave ids %q and %q, open %v, at %v", first.ID, second.ID, first.Open(), first.At)
	}

	if err := ResolveRequest(reqs, "r1", " ", "tool/1", at); err == nil {
		t.Error("ResolveRequest accepted an empty response")
	}
	if err := ResolveRequest(reqs, "r9", "done", "tool/1", at); err == nil {
		t.Error("ResolveRequest accepted an unknown id")
	}
	if err := ResolveRequest(reqs, "r1", "corrected the interval", "tool/1", at); err != nil {
		t.Fatal(err)
	}
	if err := ResolveRequest(reqs, "r1", "again", "tool/1", at); err == nil {
		t.Error("ResolveRequest resolved a request twice")
	}

	if err := SaveRequests(dir, reqs); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRequests(dir)
	if err != nil || len(loaded) != 2 {
		t.Fatalf("after saving, LoadRequests gave %v, %v", loaded, err)
	}
	if r := loaded[0]; r.Open() || r.Response != "corrected the interval" || r.ResolvedBy != "tool/1" || r.ResolvedAt == nil {
		t.Errorf("resolved request came back as %+v", r)
	}

	if _, err := WithdrawRequest(loaded, "r1"); err == nil {
		t.Error("WithdrawRequest removed a request that was already done")
	}
	loaded, err = WithdrawRequest(loaded, "r2")
	if err != nil || len(loaded) != 1 {
		t.Errorf("WithdrawRequest gave %v, %v", loaded, err)
	}
	// A new request never reuses the id of one that still exists.
	if _, next := AddRequest(loaded, "", "x", "human:me", at); next.ID != "r2" {
		t.Errorf("next id = %q, want r2", next.ID)
	}

	if err := SaveRequests(dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, RequestsFile)); !os.IsNotExist(err) {
		t.Error("saving no requests left the file behind")
	}
}
