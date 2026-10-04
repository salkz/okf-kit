package kit

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/salkz/okf-kit/internal/okf"
)

var testNow = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func read(t *testing.T, parts ...string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func write(t *testing.T, text string, parts ...string) {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(filepath.Dir(p), 0o775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o664); err != nil {
		t.Fatal(err)
	}
}

// initRepo returns a fresh repository set up at v1.0.0.
func initRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := Init(repo, "v1.0.0", testNow, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	return repo
}

func problems(t *testing.T, repo, version string, upstream bool) string {
	t.Helper()
	bundle, err := okf.Check(filepath.Join(repo, BundleDir))
	if err != nil {
		t.Fatal(err)
	}
	base, err := Check(repo, version, upstream)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(append(bundle, base...), "\n")
}

func TestInitGivesASoundBundle(t *testing.T) {
	repo := initRepo(t)
	if got := problems(t, repo, "v1.0.0", false); got != "" {
		t.Errorf("a freshly set up repository has problems:\n%s", got)
	}
	for _, f := range []string{"okf", "okf.lock", "AGENTS.md", "docker-compose.review.yml", ".claude/settings.json",
		"knowledge/index.md", "knowledge/log.md", "knowledge/checks.md", "knowledge/baseline/playbooks/make-a-change.md"} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(f))); err != nil {
			t.Errorf("init did not write %s", f)
		}
	}
	if info, _ := os.Stat(filepath.Join(repo, "okf")); info != nil && info.Mode()&0o100 == 0 {
		t.Error("the okf script is not executable")
	}
	if _, err := os.Stat(filepath.Join(repo, BundleDir, BaselineDir, "embed.go")); err == nil {
		t.Error("the baseline's Go file was copied into the repository")
	}
	if !strings.Contains(read(t, repo, "docker-compose.review.yml"), Image+":v1.0.0") {
		t.Error("the compose file does not pin the image to the kit version")
	}
	if err := Init(repo, "v1.0.0", testNow, &bytes.Buffer{}); err == nil {
		t.Error("init ran twice in one repository")
	}
}

func TestInitKeepsWhatIsThere(t *testing.T) {
	repo := t.TempDir()
	write(t, "# Mine\n\nMy instructions.\n", repo, "AGENTS.md")
	write(t, `{"model": "x", "hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "echo hi"}]}]}}`, repo, ".claude", "settings.json")
	write(t, "---\nokf_version: \"0.2\"\n---\n\n# Mine\n", repo, "knowledge", "index.md")
	var out bytes.Buffer
	if err := Init(repo, "v1.0.0", testNow, &out); err != nil {
		t.Fatal(err)
	}

	agents := read(t, repo, "AGENTS.md")
	if !strings.HasPrefix(agents, "# Mine\n\nMy instructions.\n\n"+blockStart) || !strings.HasSuffix(agents, blockEnd+"\n") {
		t.Errorf("AGENTS.md lost its own text or the block is misplaced:\n%s", agents)
	}
	settings := read(t, repo, ".claude", "settings.json")
	var parsed struct {
		Model string
		Hooks struct{ SessionStart []any }
	}
	if err := json.Unmarshal([]byte(settings), &parsed); err != nil || parsed.Model != "x" || len(parsed.Hooks.SessionStart) != 2 {
		t.Errorf("settings.json was not merged: %s", settings)
	}
	if !strings.Contains(read(t, repo, "knowledge", "index.md"), "# Mine") || !strings.Contains(out.String(), "index.md exists") {
		t.Error("an existing index.md was overwritten, or init did not say it left it alone")
	}
}

func TestCheckCatchesAChangedBaseline(t *testing.T) {
	tests := []struct {
		name   string
		change func(t *testing.T, repo string)
		want   string
	}{
		{"edited file", func(t *testing.T, repo string) {
			write(t, read(t, repo, "knowledge", "baseline", "rules", "secrets.md")+"more\n", repo, "knowledge", "baseline", "rules", "secrets.md")
		}, "baseline/rules/secrets.md was edited"},
		{"removed file", func(t *testing.T, repo string) {
			os.Remove(filepath.Join(repo, "knowledge", "baseline", "kit.json"))
		}, "baseline/kit.json is missing"},
		{"added file", func(t *testing.T, repo string) {
			write(t, "x", repo, "knowledge", "baseline", "notes.txt")
		}, "baseline/notes.txt is not part of the baseline"},
		{"no lock", func(t *testing.T, repo string) {
			os.Remove(filepath.Join(repo, LockFile))
		}, "okf.lock is missing"},
		{"hash changed in the lock to hide an edit", func(t *testing.T, repo string) {
			p := []string{repo, "knowledge", "baseline", "rules", "secrets.md"}
			write(t, read(t, p...)+"more\n", p...)
			lock, _ := ReadLock(repo)
			lock.Baseline["rules/secrets.md"] = hash([]byte(read(t, p...)))
			raw, _ := json.Marshal(lock)
			write(t, string(raw), repo, LockFile)
		}, "okf.lock does not list the baseline of v1.0.0"},
		{"required concept removed", func(t *testing.T, repo string) {
			os.Remove(filepath.Join(repo, "knowledge", "checks.md"))
		}, "checks.md is missing; the baseline needs it"},
	}
	for _, tt := range tests {
		repo := initRepo(t)
		tt.change(t, repo)
		if got := problems(t, repo, "v1.0.0", false); !strings.Contains(got, tt.want) {
			t.Errorf("%s: problems do not mention %q:\n%s", tt.name, tt.want, got)
		}
	}

	// In the kit's own repository the baseline is edited in place.
	repo := initRepo(t)
	os.Remove(filepath.Join(repo, LockFile))
	write(t, read(t, repo, "knowledge", "baseline", "index.md")+"\n", repo, "knowledge", "baseline", "index.md")
	if got := problems(t, repo, "v1.0.0", true); got != "" {
		t.Errorf("upstream mode reports problems:\n%s", got)
	}
}

func TestUpdate(t *testing.T) {
	repo := initRepo(t)
	bundle := filepath.Join(repo, BundleDir)

	// The repository was set up by an older release whose secrets rule had
	// different text, and it edited AGENTS.md around the block.
	lock, _ := ReadLock(repo)
	lock.Version, lock.Baseline["rules/secrets.md"] = "v0.9.0", "an older hash"
	raw, _ := json.Marshal(lock)
	write(t, string(raw), repo, LockFile)
	write(t, "# Mine\n\n"+strings.Replace(read(t, repo, "AGENTS.md"), "## Knowledge", "## Old heading", 1)+"\n## After\n", repo, "AGENTS.md")
	// It overrides the secrets rule and, separately, the commits rule.
	override := func(name, target string) {
		write(t, "---\ntype: Rule\ntitle: "+name+"\ndescription: Local "+name+".\noverrides: "+target+"\n"+
			"generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\n---\n\nLocal text.\n", bundle, "rules", name+".md")
	}
	override("secrets", "../baseline/rules/secrets.md")
	override("commits", "../baseline/rules/commits.md")
	write(t, "# Rules\n\n* [secrets](secrets.md) - Local secrets.\n* [commits](commits.md) - Local commits.\n", bundle, "rules", "index.md")

	var out bytes.Buffer
	if err := Update(repo, "v1.0.0", testNow, &out); err != nil {
		t.Fatal(err)
	}
	if lock, _ := ReadLock(repo); lock.Version != "v1.0.0" {
		t.Errorf("lock version after update = %q", lock.Version)
	}
	agents := read(t, repo, "AGENTS.md")
	if !strings.HasPrefix(agents, "# Mine\n\n# AGENTS.md\n\n"+blockStart) || !strings.Contains(agents, "## Knowledge") ||
		strings.Contains(agents, "## Old heading") || !strings.HasSuffix(agents, blockEnd+"\n\n## After\n") {
		t.Errorf("AGENTS.md after update:\n%s", agents)
	}

	reqs, err := okf.LoadRequests(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].Concept != "rules/secrets.md" || reqs[0].By != updater || !reqs[0].Open() ||
		!strings.Contains(reqs[0].Text, "baseline/rules/secrets.md") || !strings.Contains(reqs[0].Text, "v0.9.0") {
		t.Errorf("update should leave one request, on the override of the changed concept; got %+v", reqs)
	}

	if err := Update(t.TempDir(), "v1.0.0", testNow, &out); err == nil {
		t.Error("update ran in a repository that was never set up")
	}
}

func TestStatus(t *testing.T) {
	release := func(tag string) func() (Release, error) {
		return func() (Release, error) {
			return Release{Tag: tag, Notes: "- changed the commits rule", URL: "https://example.com/" + tag}, nil
		}
	}
	tests := []struct {
		name  string
		lock  string
		fetch func() (Release, error)
		want  []string
	}{
		{"up to date", "v1.2.0", release("v1.2.0"), []string{"v1.2.0: the baseline is up to date"}},
		{"release is older", "v1.2.0", release("v1.1.9"), []string{"up to date"}},
		{"minor update", "v1.2.0", release("v1.10.0"), []string{"v1.2.0 -> v1.10.0", "its own pull request", "changed the commits rule", "https://example.com/v1.10.0"}},
		{"major update", "v1.2.0", release("v2.0.0"), []string{"v1.2.0 -> v2.0.0", "major version", "tell the owner"}},
		{"no network", "v1.2.0", func() (Release, error) { return Release{}, errors.New("no route") }, []string{"could not check", "no route"}},
		{"development build", "dev", release("v9.0.0"), []string{"unreleased build"}},
	}
	for _, tt := range tests {
		repo := t.TempDir()
		write(t, `{"version": "`+tt.lock+`"}`, repo, LockFile)
		var out bytes.Buffer
		if err := Status(repo, &out, "", testNow, tt.fetch); err != nil {
			t.Errorf("%s: %v", tt.name, err)
		}
		for _, want := range tt.want {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s: output lacks %q:\n%s", tt.name, want, out.String())
			}
		}
	}

	var out bytes.Buffer
	if err := Status(t.TempDir(), &out, "", testNow, release("v1.0.0")); err != nil || !strings.Contains(out.String(), "okf init") {
		t.Errorf("without a lock: %q, %v", out.String(), err)
	}
}

func TestStatusUsesTheCacheForADay(t *testing.T) {
	repo := t.TempDir()
	write(t, `{"version": "v1.0.0"}`, repo, LockFile)
	cache := filepath.Join(t.TempDir(), "deep", "latest.json")
	calls := 0
	fetch := func() (Release, error) {
		calls++
		return Release{Tag: "v1.1.0"}, nil
	}
	for _, at := range []time.Time{testNow, testNow.Add(23 * time.Hour), testNow.Add(25 * time.Hour)} {
		var out bytes.Buffer
		if err := Status(repo, &out, cache, at, fetch); err != nil || !strings.Contains(out.String(), "v1.1.0") {
			t.Fatalf("status at %v: %q, %v", at, out.String(), err)
		}
	}
	if calls != 2 {
		t.Errorf("fetched %d times over 25 hours, want 2", calls)
	}
}

func TestBaselineStandsOnItsOwn(t *testing.T) {
	files, err := embedded()
	if err != nil || len(files) == 0 {
		t.Fatalf("embedded baseline: %d files, %v", len(files), err)
	}
	for p, raw := range files {
		if !strings.HasSuffix(p, ".md") || strings.HasSuffix(p, "index.md") {
			continue
		}
		c, err := okf.Parse(string(raw))
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		// A baseline concept is copied into other repositories, so it can
		// only point at things that exist everywhere: URLs.
		for _, r := range c.Resources() {
			if !strings.Contains(r, "://") {
				t.Errorf("%s: resource %q is a path; baseline concepts may only name URLs", p, r)
			}
		}
	}
}
