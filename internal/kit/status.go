package kit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Release is what Status needs to know about the newest release.
type Release struct {
	Tag   string `json:"tag_name"`
	Notes string `json:"body"`
	URL   string `json:"html_url"`
}

// cacheFor is how long a fetched release is reused, so that a check at the
// start of every session does not hit the network each time.
const cacheFor = 24 * time.Hour

// LatestRelease asks GitHub for the newest release of the kit.
func LatestRelease() (Release, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/salkz/okf-kit/releases/latest")
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var r Release
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r)
	return r, err
}

type cached struct {
	Fetched time.Time `json:"fetched"`
	Release Release   `json:"release"`
}

// cachedRelease returns the release from fetch, or from cacheFile if that
// was written less than cacheFor ago. An empty cacheFile disables caching.
func cachedRelease(cacheFile string, now time.Time, fetch func() (Release, error)) (Release, error) {
	if raw, err := os.ReadFile(cacheFile); err == nil {
		var c cached
		if json.Unmarshal(raw, &c) == nil && now.Sub(c.Fetched) < cacheFor && c.Release.Tag != "" {
			return c.Release, nil
		}
	}
	r, err := fetch()
	if err != nil {
		return Release{}, err
	}
	if cacheFile != "" {
		if raw, err := json.Marshal(cached{Fetched: now, Release: r}); err == nil {
			// A cache that cannot be written only costs a request next time.
			_ = writeFile(cacheFile, raw, 0o664)
		}
	}
	return r, nil
}

// CacheFile is where the newest release is remembered between runs.
func CacheFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "okf-kit", "latest.json")
}

// semver parses "v1.2.3". ok is false for anything else, such as a
// development build.
func semver(v string) (parts [3]int, ok bool) {
	fields := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if !strings.HasPrefix(v, "v") || len(fields) != 3 {
		return parts, false
	}
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}

// newer reports whether version a is newer than b.
func newer(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// maxNotes is how many lines of release notes Status prints.
const maxNotes = 40

// Status tells whoever starts work in repo whether its baseline is behind
// the newest release. It never fails the caller for a network problem: it
// says what it could not check and returns nil.
func Status(repo string, w io.Writer, cacheFile string, now time.Time, fetch func() (Release, error)) error {
	lock, err := ReadLock(repo)
	if err != nil {
		return err
	}
	if lock == nil {
		fmt.Fprintln(w, "okf-kit: this repository has no okf.lock; run `okf init` to set it up.")
		return nil
	}
	have, ok := semver(lock.Version)
	if !ok {
		fmt.Fprintf(w, "okf-kit: baseline is from the unreleased build %q; cannot compare it with releases.\n", lock.Version)
		return nil
	}
	latest, err := cachedRelease(cacheFile, now, fetch)
	if err != nil {
		fmt.Fprintf(w, "okf-kit %s: could not check for a newer release (%v).\n", lock.Version, err)
		return nil
	}
	want, ok := semver(latest.Tag)
	if !ok || !newer(want, have) {
		fmt.Fprintf(w, "okf-kit %s: the baseline is up to date.\n", lock.Version)
		return nil
	}

	fmt.Fprintf(w, "okf-kit: a newer baseline is available: %s -> %s.\n", lock.Version, latest.Tag)
	if want[0] > have[0] {
		fmt.Fprintln(w, "This is a major version: it removes or reverses a rule. Do not update on your own; tell the owner.")
	} else {
		fmt.Fprintf(w, "Update it in its own pull request before other work; see %s/%s/playbooks/update-baseline.md.\n", BundleDir, BaselineDir)
	}
	if notes := strings.TrimSpace(latest.Notes); notes != "" {
		lines := strings.Split(notes, "\n")
		if len(lines) > maxNotes {
			lines = append(lines[:maxNotes], "...")
		}
		fmt.Fprintf(w, "\nRelease notes for %s:\n%s\n", latest.Tag, strings.Join(lines, "\n"))
	}
	if latest.URL != "" {
		fmt.Fprintln(w, latest.URL)
	}
	return nil
}
