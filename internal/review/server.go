// Package review serves a knowledge bundle as web pages so a person can
// read each concept next to its sources and either mark it verified or
// request a change. Verifying writes a "verified" entry into the concept's
// frontmatter; a change request goes into the bundle's requests file for an
// agent to act on. Committing is left to the reviewer.
package review

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/salkz/okf-kit/internal/okf"
)

// maxSource is the largest source file the viewer will show.
const maxSource = 1 << 20

// maxRequest is the longest change request, in bytes.
const maxRequest = 4000

var reviewerID = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// ListRequests prints the open change requests, for an agent or a person
// about to act on them.
func ListRequests(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("requests", flag.ContinueOnError)
	bundle := fs.String("bundle", "knowledge", "path to the knowledge bundle")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reqs, err := okf.LoadRequests(*bundle)
	if err != nil {
		return err
	}
	open := 0
	for _, r := range reqs {
		if !r.Open() {
			continue
		}
		open++
		about := r.Concept
		if about == "" {
			about = "(the whole bundle)"
		}
		fmt.Fprintf(w, "%s  %s  requested by %s on %s\n", r.ID, about, r.By, r.At.Format("2006-01-02 15:04 UTC"))
		for _, line := range strings.Split(strings.TrimSpace(r.Text), "\n") {
			fmt.Fprintf(w, "    %s\n", line)
		}
	}
	if open == 0 {
		fmt.Fprintln(w, "No open change requests.")
	}
	return nil
}

// ResolveRequest marks one change request as done.
func ResolveRequest(w io.Writer, args []string, now time.Time) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	bundle := fs.String("bundle", "knowledge", "path to the knowledge bundle")
	id := fs.String("id", "", "id of the request, for example r1 (required)")
	by := fs.String("by", "", "who acted on it, as <tool>/<version> or human:<id> (required)")
	response := fs.String("response", "", "what was changed, for the reviewer to read (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !okf.ValidActor(*by) {
		return errors.New("-by must be <tool>/<version>, human:<id> or process:<id>")
	}
	reqs, err := okf.LoadRequests(*bundle)
	if err != nil {
		return err
	}
	if err := okf.ResolveRequest(reqs, *id, *response, *by, now); err != nil {
		return err
	}
	if err := okf.SaveRequests(*bundle, reqs); err != nil {
		return err
	}
	fmt.Fprintf(w, "Request %s is done.\n", *id)
	return nil
}

// Options says how baseline concepts are treated.
type Options struct {
	// Baseline is the folder inside the bundle that holds concepts managed
	// upstream, for example "baseline". Concepts in it are shown but can be
	// neither verified nor changed here. Empty means every concept is the
	// repository's own, as in the kit itself.
	Baseline string
	// Upstream is where a reviewer is sent to ask for a change to a
	// baseline concept.
	Upstream string
}

// Serve runs the review site until it fails.
func Serve(args []string, opts Options) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8765", "address to listen on")
	bundle := fs.String("bundle", "knowledge", "path to the knowledge bundle")
	reviewer := fs.String("reviewer", "", "your id; verifications are recorded as human:<id> (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !reviewerID.MatchString(*reviewer) {
		return errors.New("-reviewer must be set to an id of letters, digits, '.', '_' or '-'")
	}
	s, err := newServer(*bundle, "human:"+*reviewer, opts)
	if err != nil {
		return err
	}
	log.Printf("okf: serving %s on %s, verifying as %s", s.bundle, *addr, s.actor)
	srv := &http.Server{Addr: *addr, Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

// server shows one bundle and records verifications by one actor.
type server struct {
	bundle string // absolute path of the bundle
	repo   string // absolute path of the folder holding the bundle
	actor  string // "human:<id>"
	opts   Options
	now    func() time.Time
	// writing serialises changes to the requests file.
	writing sync.Mutex
}

func newServer(bundle, actor string, opts Options) (*server, error) {
	abs, err := filepath.Abs(bundle)
	if err != nil {
		return nil, err
	}
	// Resolved once, so later "is this inside the repository" checks
	// compare real paths.
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	if _, err := okf.Load(abs); err != nil {
		return nil, err
	}
	return &server{bundle: abs, repo: filepath.Dir(abs), actor: actor, opts: opts, now: time.Now}, nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleList)
	mux.HandleFunc("GET /c/{path...}", s.handleConcept)
	mux.HandleFunc("POST /verify/{path...}", s.handleVerify)
	mux.HandleFunc("POST /unverify/{path...}", s.handleVerify)
	mux.HandleFunc("POST /request/{path...}", s.handleRequest)
	mux.HandleFunc("POST /withdraw/{id}", s.handleWithdraw)
	mux.HandleFunc("GET /src/{path...}", s.handleSource)
	return mux
}

// within reports whether the absolute path p is dir or inside it.
func within(dir, p string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// target resolves a link or resource written in the concept at docPath to
// an absolute filesystem path. ok is false for URLs and page anchors.
func (s *server) target(docPath, ref string) (abs string, ok bool) {
	if strings.Contains(ref, "://") || strings.HasPrefix(ref, "mailto:") || strings.HasPrefix(ref, "#") {
		return "", false
	}
	ref, _, _ = strings.Cut(ref, "#")
	if strings.HasPrefix(ref, "/") {
		return filepath.Join(s.bundle, filepath.FromSlash(ref)), true
	}
	return filepath.Join(s.bundle, filepath.FromSlash(path.Dir(docPath)), filepath.FromSlash(ref)), true
}

// href turns a link or resource in the concept at docPath into a URL on
// this site: a concept page, a section of the list, or the source viewer.
func (s *server) href(docPath, ref string) (string, bool) {
	abs, local := s.target(docPath, ref)
	if !local {
		return ref, !strings.HasPrefix(ref, "#")
	}
	if within(s.bundle, abs) {
		rel, _ := filepath.Rel(s.bundle, abs)
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, ".md") && path.Base(rel) != "index.md" {
			return "/c/" + rel, false
		}
		return "/#" + strings.Trim(strings.TrimSuffix(rel, "index.md"), "/."), false
	}
	if within(s.repo, abs) {
		rel, _ := filepath.Rel(s.repo, abs)
		return "/src/" + filepath.ToSlash(rel), false
	}
	return "#", false
}

// viewable returns the repository files the source viewer may show, keyed
// by slash-separated path relative to the repository. Only what the bundle
// itself points at is shown: files named as a resource or linked from a
// concept, and the files directly inside such a folder. Everything else in
// the working tree, which can hold secrets, stays private.
func (s *server) viewable(docs []okf.Doc) map[string]bool {
	out := map[string]bool{}
	add := func(abs string) {
		real, err := filepath.EvalSymlinks(abs)
		if err != nil || !within(s.repo, real) || within(s.bundle, real) {
			return
		}
		info, err := os.Stat(real)
		if err != nil {
			return
		}
		files := []string{real}
		if info.IsDir() {
			files = nil
			entries, _ := os.ReadDir(real)
			for _, e := range entries {
				if e.Type().IsRegular() {
					files = append(files, filepath.Join(real, e.Name()))
				}
			}
		}
		for _, f := range files {
			if rel, err := filepath.Rel(s.repo, f); err == nil {
				out[filepath.ToSlash(rel)] = true
			}
		}
	}
	for _, d := range docs {
		refs := d.Resources()
		for _, m := range linkRe.FindAllStringSubmatch(d.Body, -1) {
			refs = append(refs, m[2])
		}
		for _, ref := range refs {
			if abs, local := s.target(d.Path, ref); local {
				add(abs)
			}
		}
	}
	return out
}

// Review states of a concept, as shown on its badge.
const (
	stateVerified   = "verified"
	stateUnverified = "unverified"
	stateOutdated   = "changed since it was verified"
	stateRequested  = "change requested"
	stateUpdated    = "updated, review again"
	stateBaseline   = "baseline"
)

// managed reports whether a concept belongs to the upstream baseline, which
// is reviewed where it is maintained, not in this repository.
func (s *server) managed(d okf.Doc) bool {
	return s.opts.Baseline != "" && strings.HasPrefix(d.Path, s.opts.Baseline+"/")
}

// stateOf is state, except that baseline concepts are not up for review.
func (s *server) stateOf(d okf.Doc, reqs []okf.Request) string {
	if s.managed(d) {
		return stateBaseline
	}
	return state(d, reqs)
}

// state says where a concept stands in the review, given the change
// requests about it.
func state(d okf.Doc, reqs []okf.Request) string {
	human, verified := d.HumanVerified()
	answered := false
	for _, r := range reqs {
		if r.Open() {
			return stateRequested
		}
		// An answer counts until a person has verified the result.
		if r.ResolvedAt != nil && (!verified || r.ResolvedAt.After(human.At)) {
			answered = true
		}
	}
	switch {
	case answered:
		// Also when the concept itself did not change: the reviewer
		// should see an answer that says why not.
		return stateUpdated
	case !d.NeedsReview():
		return stateVerified
	case verified:
		return stateOutdated
	}
	return stateUnverified
}

// reviewable reports whether a concept is waiting for the reviewer, rather
// than verified or waiting for an agent to act on a request.
func reviewable(st string) bool {
	return st != stateVerified && st != stateRequested && st != stateBaseline
}

// byConcept groups change requests by the concept they are about; requests
// about the whole bundle are under "".
func byConcept(reqs []okf.Request) map[string][]okf.Request {
	out := map[string][]okf.Request{}
	for _, r := range reqs {
		out[r.Concept] = append(out[r.Concept], r)
	}
	return out
}

// find returns the concept with the given bundle path and its position.
func find(docs []okf.Doc, p string) (int, bool) {
	for i, d := range docs {
		if d.Path == p {
			return i, true
		}
	}
	return 0, false
}

// nextToReview returns the path of the first concept after position i that
// is waiting for the reviewer, wrapping around, or "" if none is. Pass -1
// to search from the start.
func (s *server) nextToReview(docs []okf.Doc, reqs map[string][]okf.Request, i int) string {
	for n := 1; n <= len(docs); n++ {
		j := (i + n) % len(docs)
		if d := docs[j]; j != i && reviewable(s.stateOf(d, reqs[d.Path])) {
			return d.Path
		}
	}
	return ""
}

func (s *server) load(w http.ResponseWriter) ([]okf.Doc, map[string][]okf.Request, bool) {
	docs, err := okf.Load(s.bundle)
	if err != nil {
		http.Error(w, "The bundle could not be read: "+err.Error(), http.StatusInternalServerError)
		return nil, nil, false
	}
	reqs, err := okf.LoadRequests(s.bundle)
	if err != nil {
		http.Error(w, "The change requests could not be read: "+err.Error(), http.StatusInternalServerError)
		return nil, nil, false
	}
	return docs, byConcept(reqs), true
}

func (s *server) page(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Nothing on a page needs scripts, so none are allowed to run.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'")
	if err := pages.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("okf: %v", err)
	}
}

type listRow struct {
	Path, Title, Type, Description, State string
}

type listSection struct {
	Name string
	Rows []listRow
}

func (s *server) handleList(w http.ResponseWriter, r *http.Request) {
	docs, reqs, ok := s.load(w)
	if !ok {
		return
	}
	var sections []listSection
	verified, waiting, own := 0, 0, 0
	for _, d := range docs {
		name := path.Dir(d.Path)
		if name == "." {
			name = ""
		}
		if len(sections) == 0 || sections[len(sections)-1].Name != name {
			sections = append(sections, listSection{Name: name})
		}
		st := s.stateOf(d, reqs[d.Path])
		sec := &sections[len(sections)-1]
		sec.Rows = append(sec.Rows, listRow{d.Path, title(d), d.Fields["type"], d.Fields["description"], st})
		switch st {
		case stateVerified:
			verified++
			own++
		case stateBaseline:
		default:
			own++
		}
	}
	for _, rs := range reqs {
		for _, r := range rs {
			if r.Open() {
				waiting++
			}
		}
	}
	percent := 0
	if own > 0 {
		percent = verified * 100 / own
	}
	s.page(w, "list", map[string]any{
		"Sections": sections, "Verified": verified, "Total": own, "Percent": percent,
		"Next": s.nextToReview(docs, reqs, -1), "Actor": s.actor, "Waiting": waiting,
		"Requests": reqs[""], "MaxRequest": maxRequest,
	})
}

// title is the concept's display name.
func title(d okf.Doc) string {
	if t := d.Fields["title"]; t != "" {
		return t
	}
	return strings.TrimSuffix(path.Base(d.Path), ".md")
}

type sourceLink struct {
	ID, Title, Href string
	External        bool
}

func (s *server) handleConcept(w http.ResponseWriter, r *http.Request) {
	docs, reqs, ok := s.load(w)
	if !ok {
		return
	}
	i, found := find(docs, r.PathValue("path"))
	if !found {
		http.NotFound(w, r)
		return
	}
	d := docs[i]
	st := s.stateOf(d, reqs[d.Path])
	link := func(ref string) (string, bool) { return s.href(d.Path, ref) }

	var sources []sourceLink
	for _, src := range d.Sources {
		href, external := link(src.Resource)
		label := src.Title
		if label == "" {
			label = src.Resource
		}
		sources = append(sources, sourceLink{src.ID, label, href, external})
	}
	var resource *sourceLink
	if res := d.Fields["resource"]; res != "" {
		href, external := link(res)
		resource = &sourceLink{Title: res, Href: href, External: external}
	}
	var overrides *sourceLink
	if o := d.Fields["overrides"]; o != "" {
		href, _ := link(o)
		overrides = &sourceLink{Title: o, Href: href}
	}
	// Local concepts that replace this one, if it is a baseline concept.
	var overriddenBy []sourceLink
	for _, other := range docs {
		o := other.Fields["overrides"]
		if o == "" {
			continue
		}
		if href, _ := s.href(other.Path, o); href == "/c/"+d.Path {
			overriddenBy = append(overriddenBy, sourceLink{Title: title(other), Href: "/c/" + other.Path})
		}
	}
	human, hasHuman := d.HumanVerified()
	mine := false
	for _, ev := range d.Verified {
		mine = mine || ev.By == s.actor
	}
	s.page(w, "concept", map[string]any{
		"Path": d.Path, "Title": title(d), "Type": d.Fields["type"], "Description": d.Fields["description"],
		"Tags": strings.Trim(d.Fields["tags"], "[]"), "Status": d.Fields["status"],
		"Generated": d.Generated, "Verified": d.Verified, "Resource": resource, "Sources": sources,
		"Body":  template.HTML(render(d.Body, link)),
		"State": st, "Verifiable": reviewable(st), "Requested": st == stateRequested,
		"Managed": st == stateBaseline, "Upstream": s.opts.Upstream,
		"Overrides": overrides, "OverriddenBy": overriddenBy,
		"HasHuman": hasHuman, "Human": human,
		"Mine": mine, "Actor": s.actor, "Next": s.nextToReview(docs, reqs, i),
		"Position": i + 1, "Total": len(docs),
		"Requests": reqs[d.Path], "MaxRequest": maxRequest,
	})
}

// sameOrigin rejects a form post that another website made the browser
// send, which is the one way a page elsewhere could verify concepts.
func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		return err == nil && u.Host == r.Host
	}
	return true
}

func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "This request did not come from the review site.", http.StatusForbidden)
		return
	}
	docs, reqs, ok := s.load(w)
	if !ok {
		return
	}
	i, found := find(docs, r.PathValue("path"))
	if !found {
		http.NotFound(w, r)
		return
	}
	if s.managed(docs[i]) {
		http.Error(w, "Baseline concepts are verified where the baseline is maintained.", http.StatusForbidden)
		return
	}
	// The path comes from the loaded bundle, never straight from the URL.
	file := filepath.Join(s.bundle, filepath.FromSlash(docs[i].Path))
	verify := strings.HasPrefix(r.URL.Path, "/verify/")
	if err := s.rewrite(file, verify); err != nil {
		http.Error(w, "The concept could not be updated: "+err.Error(), http.StatusInternalServerError)
		return
	}
	dest := "/c/" + docs[i].Path
	if verify {
		log.Printf("okf: %s verified %s", s.actor, docs[i].Path)
		if next := s.nextToReview(docs, reqs, i); next != "" {
			dest = "/c/" + next
		} else {
			dest = "/"
		}
	} else {
		log.Printf("okf: %s removed their verification of %s", s.actor, docs[i].Path)
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// rewrite adds or removes this server's verification in a concept file,
// replacing the file in one step.
func (s *server) rewrite(file string, verify bool) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	var text string
	if verify {
		text, err = okf.Verify(string(raw), s.actor, s.now())
	} else {
		text, err = okf.Unverify(string(raw), s.actor)
	}
	if err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// handleRequest records a change request about a concept, or about the
// whole bundle when the path is empty.
func (s *server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "This request did not come from the review site.", http.StatusForbidden)
		return
	}
	docs, _, ok := s.load(w)
	if !ok {
		return
	}
	concept := r.PathValue("path")
	i, found := find(docs, concept)
	if concept != "" && !found {
		http.NotFound(w, r)
		return
	}
	if found && s.managed(docs[i]) {
		http.Error(w, "A change to a baseline concept is requested where the baseline is maintained.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4*maxRequest)
	text := strings.TrimSpace(strings.ReplaceAll(r.FormValue("text"), "\r\n", "\n"))
	if text == "" || len(text) > maxRequest {
		http.Error(w, fmt.Sprintf("Describe the change in 1 to %d characters.", maxRequest), http.StatusBadRequest)
		return
	}

	s.writing.Lock()
	defer s.writing.Unlock()
	reqs, err := okf.LoadRequests(s.bundle)
	if err == nil {
		var req okf.Request
		reqs, req = okf.AddRequest(reqs, concept, text, s.actor, s.now())
		if err = okf.SaveRequests(s.bundle, reqs); err == nil {
			log.Printf("okf: %s requested a change (%s) to %q", s.actor, req.ID, concept)
		}
	}
	if err != nil {
		http.Error(w, "The request could not be saved: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, back(concept), http.StatusSeeOther)
}

// handleWithdraw removes an open change request.
func (s *server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "This request did not come from the review site.", http.StatusForbidden)
		return
	}
	s.writing.Lock()
	defer s.writing.Unlock()
	reqs, err := okf.LoadRequests(s.bundle)
	if err != nil {
		http.Error(w, "The change requests could not be read: "+err.Error(), http.StatusInternalServerError)
		return
	}
	id, concept := r.PathValue("id"), ""
	for _, req := range reqs {
		if req.ID == id {
			concept = req.Concept
		}
	}
	if reqs, err = okf.WithdrawRequest(reqs, id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := okf.SaveRequests(s.bundle, reqs); err != nil {
		http.Error(w, "The request could not be removed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("okf: %s withdrew request %s", s.actor, id)
	http.Redirect(w, r, back(concept), http.StatusSeeOther)
}

// back is the page a change request belongs to.
func back(concept string) string {
	if concept == "" {
		return "/#requests"
	}
	return "/c/" + concept + "#requests"
}

type sourceLine struct {
	N    int
	Text string
}

func (s *server) handleSource(w http.ResponseWriter, r *http.Request) {
	docs, _, ok := s.load(w)
	if !ok {
		return
	}
	allowed := s.viewable(docs)
	rel := path.Clean(r.PathValue("path"))

	if !allowed[rel] {
		// A folder is shown as the list of its viewable files.
		var files []string
		for f := range allowed {
			if path.Dir(f) == rel {
				files = append(files, f)
			}
		}
		if len(files) == 0 {
			http.NotFound(w, r)
			return
		}
		sort.Strings(files)
		s.page(w, "folder", map[string]any{"Path": rel, "Files": files})
		return
	}

	raw, err := os.ReadFile(filepath.Join(s.repo, filepath.FromSlash(rel)))
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	note := ""
	if len(raw) > maxSource {
		raw, note = raw[:maxSource], fmt.Sprintf("Only the first %d KiB are shown.", maxSource>>10)
	}
	var lines []sourceLine
	for i, l := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		lines = append(lines, sourceLine{i + 1, l})
	}
	s.page(w, "source", map[string]any{"Path": rel, "Lines": lines, "Note": note})
}
