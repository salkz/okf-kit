// Package kit puts the baseline and the files around it into a repository
// and keeps them at the version of the running tool: init, update, the
// integrity check and the check for a newer release.
package kit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/salkz/okf-kit/internal/okf"
	"github.com/salkz/okf-kit/knowledge/baseline"
)

const (
	// Module is the Go module path of the kit.
	Module = "github.com/salkz/okf-kit"
	// Image is the kit's container image, without a tag.
	Image = "ghcr.io/salkz/okf-kit"
	// Upstream is where the kit and its baseline are maintained.
	Upstream = "https://github.com/salkz/okf-kit"

	// BundleDir is the bundle folder in a repository.
	BundleDir = "knowledge"
	// BaselineDir is the baseline folder inside the bundle.
	BaselineDir = "baseline"
	// LockFile records which kit version a repository's baseline is from.
	LockFile = "okf.lock"

	// updater is the actor that leaves change requests during an update.
	updater = "process:okf-update"
)

// Lock is the content of okf.lock.
type Lock struct {
	Kit     string `json:"kit"`
	Version string `json:"version"`
	// Baseline maps each baseline file, relative to the baseline folder,
	// to the SHA-256 of its content.
	Baseline map[string]string `json:"baseline"`
}

// ReadLock reads okf.lock in repo. It returns nil if there is none.
func ReadLock(repo string) (*Lock, error) {
	raw, err := os.ReadFile(filepath.Join(repo, LockFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", LockFile, err)
	}
	return &l, nil
}

func hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// embedded returns the baseline shipped in this binary: path to content.
func embedded() (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(baseline.Files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := baseline.Files.ReadFile(p)
		files[p] = b
		return err
	})
	return files, err
}

// manifest hashes a set of files.
func manifest(files map[string][]byte) map[string]string {
	m := make(map[string]string, len(files))
	for p, b := range files {
		m[p] = hash(b)
	}
	return m
}

// onDisk hashes the files of the baseline folder in repo.
func onDisk(repo string) (map[string]string, error) {
	root := filepath.Join(repo, BundleDir, BaselineDir)
	m := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		m[filepath.ToSlash(rel)] = hash(b)
		return err
	})
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	return m, err
}

// required lists the local concepts the baseline expects, relative to the
// bundle.
func required(repo string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(repo, BundleDir, BaselineDir, "kit.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var k struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, fmt.Errorf("%s/kit.json: %w", BaselineDir, err)
	}
	return k.Required, nil
}

// Check reports what is wrong with the baseline in repo, one line per
// problem. In a repository that uses the kit, the baseline must be exactly
// what okf.lock records. upstream is for the kit's own repository, where
// the baseline is edited and there is no lock. version is the running tool.
func Check(repo, version string, upstream bool) ([]string, error) {
	var problems []string
	lock, err := ReadLock(repo)
	if err != nil {
		return nil, err
	}
	disk, err := onDisk(repo)
	if err != nil {
		return nil, err
	}
	switch {
	case upstream:
	case lock == nil:
		problems = append(problems, LockFile+" is missing; run `okf init`, or pass -upstream in the kit's own repository")
	default:
		for _, p := range sortedKeys(lock.Baseline, disk) {
			want, locked := lock.Baseline[p]
			got, present := disk[p]
			switch {
			case !present:
				problems = append(problems, fmt.Sprintf("%s/%s is missing; baseline files are restored with `okf update`", BaselineDir, p))
			case !locked:
				problems = append(problems, fmt.Sprintf("%s/%s is not part of the baseline; local concepts go outside %s/", BaselineDir, p, BaselineDir))
			case want != got:
				problems = append(problems, fmt.Sprintf("%s/%s was edited; baseline files change only through `okf update`", BaselineDir, p))
			}
		}
		// A lock that claims this release must list this release's files,
		// or the hashes in it were changed by hand.
		if files, err := embedded(); err == nil && lock.Version == version && !sameManifest(lock.Baseline, manifest(files)) {
			problems = append(problems, fmt.Sprintf("%s does not list the baseline of %s; run `okf update`", LockFile, version))
		}
	}

	need, err := required(repo)
	if err != nil {
		return nil, err
	}
	for _, r := range need {
		if _, err := os.Stat(filepath.Join(repo, BundleDir, filepath.FromSlash(r))); err != nil {
			problems = append(problems, fmt.Sprintf("%s is missing; the baseline needs it, see %s/playbooks/bootstrap-knowledge.md", r, BaselineDir))
		}
	}
	return problems, nil
}

func sameManifest(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func sortedKeys(maps ...map[string]string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, m := range maps {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// writeFile writes a file and the folders above it.
func writeFile(p string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o775); err != nil {
		return err
	}
	return os.WriteFile(p, data, mode)
}

// writeIfMissing writes a file only if nothing is there yet, and says
// whether it did.
func writeIfMissing(p string, data []byte) (bool, error) {
	if _, err := os.Stat(p); err == nil {
		return false, nil
	}
	return true, writeFile(p, data, 0o664)
}

// sync makes the managed files in repo match this binary: the baseline
// folder, the lock, the okf script, the review compose file, the block in
// AGENTS.md and the session hook.
func sync(repo, version string, w io.Writer) error {
	files, err := embedded()
	if err != nil {
		return err
	}
	base := filepath.Join(repo, BundleDir, BaselineDir)
	if err := os.RemoveAll(base); err != nil {
		return err
	}
	for p, b := range files {
		if err := writeFile(filepath.Join(base, filepath.FromSlash(p)), b, 0o664); err != nil {
			return err
		}
	}
	lock, err := json.MarshalIndent(Lock{Kit: Module, Version: version, Baseline: manifest(files)}, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(repo, LockFile), append(lock, '\n'), 0o664); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(repo, "okf"), []byte(script), 0o775); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(repo, "docker-compose.review.yml"), []byte(composeFile(version)), 0o664); err != nil {
		return err
	}
	if err := writeAgentsBlock(filepath.Join(repo, "AGENTS.md")); err != nil {
		return err
	}
	if note, err := writeSessionHook(filepath.Join(repo, ".claude", "settings.json")); err != nil {
		return err
	} else if note != "" {
		fmt.Fprintln(w, note)
	}
	return nil
}

// Init sets a repository up with the baseline and an empty local bundle.
// It leaves existing local concepts alone.
func Init(repo, version string, now time.Time, w io.Writer) error {
	if lock, err := ReadLock(repo); err != nil {
		return err
	} else if lock != nil {
		return fmt.Errorf("this repository already uses okf-kit %s; run `okf update` instead", lock.Version)
	}
	if err := sync(repo, version, w); err != nil {
		return err
	}
	bundle := filepath.Join(repo, BundleDir)
	stamp := now.UTC().Format("2006-01-02T15:04:05Z")
	made, err := writeIfMissing(filepath.Join(bundle, "checks.md"), []byte(fmt.Sprintf(checksStub, version, stamp)))
	if err != nil {
		return err
	}
	if made {
		fmt.Fprintf(w, "Wrote %s/checks.md as a draft: fill in this repository's checks.\n", BundleDir)
	}
	if made, err = writeIfMissing(filepath.Join(bundle, "index.md"), []byte(rootIndex)); err != nil {
		return err
	} else if !made {
		fmt.Fprintf(w, "%s/index.md exists: add entries for baseline/ and checks.md to it; `okf check` says what is missing.\n", BundleDir)
	}
	if _, err := writeIfMissing(filepath.Join(bundle, "log.md"), []byte(fmt.Sprintf(logStub, now.UTC().Format("2006-01-02"), version))); err != nil {
		return err
	}
	fmt.Fprintf(w, "Set up okf-kit %s. Next: read %s/%s/playbooks/bootstrap-knowledge.md.\n", version, BundleDir, BaselineDir)
	return nil
}

// Update moves a repository's managed files to the version of this binary.
// Local concepts that override a baseline concept whose text changed get a
// change request, so that someone looks at the override again.
func Update(repo, version string, now time.Time, w io.Writer) error {
	old, err := ReadLock(repo)
	if err != nil {
		return err
	}
	if old == nil {
		return errors.New("this repository does not use okf-kit yet; run `okf init`")
	}
	if err := sync(repo, version, w); err != nil {
		return err
	}
	files, err := embedded()
	if err != nil {
		return err
	}
	changed := map[string]bool{}
	current := manifest(files)
	for _, p := range sortedKeys(old.Baseline, current) {
		if old.Baseline[p] != current[p] {
			changed[p] = true
		}
	}
	fmt.Fprintf(w, "Baseline %s -> %s: %d file(s) changed.\n", old.Version, version, len(changed))
	if len(changed) == 0 {
		return nil
	}

	bundle := filepath.Join(repo, BundleDir)
	docs, err := okf.Load(bundle)
	if err != nil {
		return err
	}
	reqs, err := okf.LoadRequests(bundle)
	if err != nil {
		return err
	}
	flagged := 0
	for _, d := range docs {
		o := d.Fields["overrides"]
		if o == "" || strings.HasPrefix(d.Path, BaselineDir+"/") {
			continue
		}
		target := path.Join(path.Dir(d.Path), o)
		if strings.HasPrefix(o, "/") {
			target = strings.TrimPrefix(o, "/")
		}
		rel, inBaseline := strings.CutPrefix(target, BaselineDir+"/")
		if !inBaseline || !changed[rel] {
			continue
		}
		text := fmt.Sprintf("The baseline concept this overrides, %s, changed between %s and %s. Read the new text and check that this override still makes sense.", target, old.Version, version)
		reqs, _ = okf.AddRequest(reqs, d.Path, text, updater, now)
		flagged++
		fmt.Fprintf(w, "  %s overrides %s: left a change request.\n", d.Path, target)
	}
	if flagged > 0 {
		return okf.SaveRequests(bundle, reqs)
	}
	return nil
}
