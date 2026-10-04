package review

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/salkz/okf-kit/internal/okf"
)

// newTestServer builds a small repository with a bundle of two concepts, a
// source file they point at, and a secret file nothing points at.
func newTestServer(t *testing.T) (*server, string) {
	t.Helper()
	repo := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o775); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o664); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/app/app.go", "package app\n\nfunc Run() {}\n")
	write("config.env", "MAM_ID=secret\n")
	write("knowledge/index.md", "# Start\n\n* [Overview](overview.md) - The overview.\n* [Rules](rules/) - Rules.\n")
	write("knowledge/overview.md", "---\ntype: Project\ntitle: Overview\ndescription: The overview.\n"+
		"generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\nsources:\n  - id: app\n    resource: ../internal/app/app.go\n    title: app.go\n---\n\n"+
		"# Shape\n\nSee the [rule](rules/safe.md) and the code.[^app]\n\n[^app]: app.go\n")
	write("knowledge/rules/safe.md", "---\ntype: Rule\ntitle: Stay safe\ndescription: A rule.\n"+
		"generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\n---\n\n# Rule\n\nBe safe. Back to the [overview](../overview.md).\n")
	s, err := newServer(filepath.Join(repo, "knowledge"), "human:tester", Options{})
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC) }
	return s, repo
}

func do(t *testing.T, s *server, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	return rec
}

func TestListAndConceptPages(t *testing.T) {
	s, _ := newTestServer(t)

	list := do(t, s, "GET", "/", nil)
	body := list.Body.String()
	for _, want := range []string{"0 of 2", `href="/c/overview.md"`, `href="/c/rules/safe.md"`, "unverified", "human:tester"} {
		if list.Code != http.StatusOK || !strings.Contains(body, want) {
			t.Errorf("list page (status %d) lacks %q", list.Code, want)
		}
	}

	page := do(t, s, "GET", "/c/overview.md", nil)
	body = page.Body.String()
	for _, want := range []string{
		"<h1>Overview</h1>", "<h2>Shape</h2>",
		`<a href="/c/rules/safe.md">rule</a>`,           // link to another concept
		`<a href="/src/internal/app/app.go">app.go</a>`, // source goes to the viewer
		`action="/verify/overview.md"`, "nobody yet",
	} {
		if page.Code != http.StatusOK || !strings.Contains(body, want) {
			t.Errorf("concept page (status %d) lacks %q", page.Code, want)
		}
	}
	if do(t, s, "GET", "/c/missing.md", nil).Code != http.StatusNotFound {
		t.Error("a missing concept is not a 404")
	}
}

func TestVerifyAndUnverify(t *testing.T) {
	s, repo := newTestServer(t)
	file := filepath.Join(repo, "knowledge", "overview.md")
	before, _ := os.ReadFile(file)

	rec := do(t, s, "POST", "/verify/overview.md", nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/c/rules/safe.md" {
		t.Fatalf("verify answered %d to %q, want a redirect to the next unverified concept", rec.Code, rec.Header().Get("Location"))
	}
	after, _ := os.ReadFile(file)
	const entry = "verified: { by: human:tester, at: 2026-10-04T12:00:00Z }\n"
	if want := strings.Replace(string(before), "sources:\n", entry+"sources:\n", 1); string(after) != want {
		t.Errorf("file after verifying:\n%s\nwant:\n%s", after, want)
	}
	if body := do(t, s, "GET", "/", nil).Body.String(); !strings.Contains(body, "1 of 2") {
		t.Error("the list does not count the verification")
	}
	if body := do(t, s, "GET", "/c/overview.md", nil).Body.String(); !strings.Contains(body, `action="/unverify/overview.md"`) {
		t.Error("a verified concept does not offer to remove the verification")
	}

	// The last one sends the reviewer back to the list.
	if rec := do(t, s, "POST", "/verify/rules/safe.md", nil); rec.Header().Get("Location") != "/" {
		t.Errorf("verifying the last concept redirects to %q, want /", rec.Header().Get("Location"))
	}

	rec = do(t, s, "POST", "/unverify/overview.md", nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/c/overview.md" {
		t.Errorf("unverify answered %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	if restored, _ := os.ReadFile(file); string(restored) != string(before) {
		t.Errorf("file after removing the verification differs from the original:\n%s", restored)
	}
	if _, err := os.Stat(file + ".tmp"); !os.IsNotExist(err) {
		t.Error("a temporary file was left behind")
	}
}

func TestVerifyRejectsOtherSitesAndMethods(t *testing.T) {
	s, repo := newTestServer(t)
	file := filepath.Join(repo, "knowledge", "overview.md")
	before, _ := os.ReadFile(file)

	tests := []struct {
		name   string
		method string
		header http.Header
		want   int
	}{
		{"post from another origin", "POST", http.Header{"Origin": {"https://evil.example"}}, http.StatusForbidden},
		{"cross-site fetch metadata", "POST", http.Header{"Sec-Fetch-Site": {"cross-site"}}, http.StatusForbidden},
		{"a link cannot verify", "GET", nil, http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		if rec := do(t, s, tt.method, "/verify/overview.md", tt.header); rec.Code != tt.want {
			t.Errorf("%s: status %d, want %d", tt.name, rec.Code, tt.want)
		}
	}
	if after, _ := os.ReadFile(file); string(after) != string(before) {
		t.Error("a rejected request changed the file")
	}
	// The same origin is accepted.
	if rec := do(t, s, "POST", "/verify/overview.md", http.Header{"Origin": {"http://example.com"}, "Sec-Fetch-Site": {"same-origin"}}); rec.Code != http.StatusSeeOther {
		t.Errorf("same-origin post: status %d", rec.Code)
	}
	if rec := do(t, s, "POST", "/verify/"+url.PathEscape("../config.env"), nil); rec.Code == http.StatusSeeOther {
		t.Error("a path outside the bundle was accepted")
	}
}

func TestSourceViewerShowsOnlyWhatTheBundleNames(t *testing.T) {
	s, _ := newTestServer(t)

	ok := do(t, s, "GET", "/src/internal/app/app.go", nil)
	if ok.Code != http.StatusOK || !strings.Contains(ok.Body.String(), "func Run() {}") {
		t.Errorf("named source: status %d", ok.Code)
	}
	for _, target := range []string{
		"/src/config.env",            // in the repository, but nothing points at it
		"/src/knowledge/overview.md", // the bundle itself is not source
		"/src/internal/app/../../config.env",
		"/src/" + url.PathEscape("../../etc/passwd"),
	} {
		rec := do(t, s, "GET", target, nil)
		if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("%s: status %d, want it refused", target, rec.Code)
		}
	}
}

// post sends a form the way the pages do.
func post(t *testing.T, s *server, target string, form url.Values, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	return rec
}

func TestChangeRequestRoundTrip(t *testing.T) {
	s, repo := newTestServer(t)
	bundle := filepath.Join(repo, "knowledge")
	text := "Line one is wrong.\r\n<b>Fix</b> the shape section."

	// 1. The reviewer asks for a change.
	rec := post(t, s, "/request/overview.md", url.Values{"text": {text}}, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/c/overview.md#requests" {
		t.Fatalf("request answered %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	reqs, err := okf.LoadRequests(bundle)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("requests file holds %v, %v", reqs, err)
	}
	want := okf.Request{ID: "r1", Concept: "overview.md", Text: "Line one is wrong.\n<b>Fix</b> the shape section.",
		By: "human:tester", At: s.now(), Status: okf.RequestOpen}
	if reqs[0] != want {
		t.Errorf("stored request = %+v, want %+v", reqs[0], want)
	}

	// 2. The concept waits for an agent and is skipped by the review order.
	page := do(t, s, "GET", "/c/overview.md", nil).Body.String()
	for _, wantText := range []string{"change requested", "waiting for an agent", "&lt;b&gt;Fix&lt;/b&gt;", `action="/withdraw/r1"`} {
		if !strings.Contains(page, wantText) {
			t.Errorf("concept page with an open request lacks %q", wantText)
		}
	}
	if strings.Contains(page, `action="/verify/overview.md"`) {
		t.Error("a concept with an open request still offers Verify")
	}
	list := do(t, s, "GET", "/", nil).Body.String()
	if !strings.Contains(list, "1 change request waiting") || !strings.Contains(list, `href="/c/rules/safe.md">Review the next concept`) {
		t.Error("the list does not show the waiting request or still sends the reviewer to that concept")
	}

	// 3. An agent updates the concept and resolves the request.
	file := filepath.Join(bundle, "overview.md")
	raw, _ := os.ReadFile(file)
	updated := strings.Replace(string(raw), "at: 2026-10-01T00:00:00Z", "at: 2026-10-04T13:00:00Z", 1)
	if err := os.WriteFile(file, []byte(updated), 0o664); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	args := []string{"-bundle", bundle, "-id", "r1", "-by", "tool/1", "-response", "Rewrote the shape section."}
	if err := ResolveRequest(&out, args, s.now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	// 4. It comes back for review, with the answer shown.
	page = do(t, s, "GET", "/c/overview.md", nil).Body.String()
	for _, wantText := range []string{"updated, review again", "Rewrote the shape section.", "tool/1", `action="/verify/overview.md"`} {
		if !strings.Contains(page, wantText) {
			t.Errorf("concept page after the answer lacks %q", wantText)
		}
	}

	// 5. Verifying closes the loop; the answered request stays as a record.
	s.now = func() time.Time { return time.Date(2026, 10, 4, 15, 0, 0, 0, time.UTC) }
	do(t, s, "POST", "/verify/overview.md", nil)
	page = do(t, s, "GET", "/c/overview.md", nil).Body.String()
	if !strings.Contains(page, `<span class="badge ok">verified</span>`) || !strings.Contains(page, "Rewrote the shape section.") {
		t.Error("after verifying, the concept is not shown as verified with its request history")
	}
}

func TestState(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	answered := func(d int) okf.Request {
		at := day(d)
		return okf.Request{Status: okf.RequestDone, ResolvedAt: &at}
	}
	human := []okf.Event{{By: "human:me", At: day(5)}}
	tests := []struct {
		name      string
		generated int
		verified  []okf.Event
		reqs      []okf.Request
		want      string
	}{
		{"new", 1, nil, nil, stateUnverified},
		{"verified", 1, human, nil, stateVerified},
		{"rewritten after verification", 6, human, nil, stateOutdated},
		{"open request wins", 1, human, []okf.Request{{Status: okf.RequestOpen}}, stateRequested},
		{"answered, never verified", 2, nil, []okf.Request{answered(2)}, stateUpdated},
		{"answered after verification", 6, human, []okf.Request{answered(6)}, stateUpdated},
		{"answered, then verified", 4, human, []okf.Request{answered(4)}, stateVerified},
		{"answered without changing a verified concept", 1, human, []okf.Request{answered(6)}, stateUpdated},
	}
	for _, tt := range tests {
		d := okf.Doc{Concept: okf.Concept{Generated: okf.Event{By: "tool/1", At: day(tt.generated)}, Verified: tt.verified}}
		if got := state(d, tt.reqs); got != tt.want {
			t.Errorf("%s: state = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRequestRejections(t *testing.T) {
	s, repo := newTestServer(t)
	tests := []struct {
		name   string
		target string
		text   string
		header http.Header
		want   int
	}{
		{"empty text", "/request/overview.md", "  \n ", nil, http.StatusBadRequest},
		{"too long", "/request/overview.md", strings.Repeat("x", maxRequest+1), nil, http.StatusBadRequest},
		{"unknown concept", "/request/missing.md", "fix", nil, http.StatusNotFound},
		{"another website", "/request/overview.md", "fix", http.Header{"Origin": {"https://evil.example"}}, http.StatusForbidden},
		{"withdraw from another website", "/withdraw/r1", "", http.Header{"Sec-Fetch-Site": {"cross-site"}}, http.StatusForbidden},
		{"withdraw an unknown request", "/withdraw/r9", "", nil, http.StatusNotFound},
	}
	for _, tt := range tests {
		if rec := post(t, s, tt.target, url.Values{"text": {tt.text}}, tt.header); rec.Code != tt.want {
			t.Errorf("%s: status %d, want %d", tt.name, rec.Code, tt.want)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, "knowledge", okf.RequestsFile)); !os.IsNotExist(err) {
		t.Error("a rejected request created the requests file")
	}
}

func TestBundleRequestAndWithdraw(t *testing.T) {
	s, repo := newTestServer(t)
	bundle := filepath.Join(repo, "knowledge")

	if rec := post(t, s, "/request/", url.Values{"text": {"add a concept about releases"}}, nil); rec.Header().Get("Location") != "/#requests" {
		t.Fatalf("bundle request answered %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	var out bytes.Buffer
	if err := ListRequests(&out, []string{"-bundle", bundle}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"r1", "(the whole bundle)", "human:tester", "    add a concept about releases"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("requests listing lacks %q:\n%s", want, out.String())
		}
	}
	if body := do(t, s, "GET", "/", nil).Body.String(); !strings.Contains(body, "add a concept about releases") {
		t.Error("the list page does not show the bundle request")
	}

	if rec := post(t, s, "/withdraw/r1", nil, nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("withdraw answered %d", rec.Code)
	}
	out.Reset()
	if err := ListRequests(&out, []string{"-bundle", bundle}); err != nil || !strings.Contains(out.String(), "No open change requests") {
		t.Errorf("after withdrawing, listing gave %q, %v", out.String(), err)
	}
}

func TestResolveNeedsAnActorAndAResponse(t *testing.T) {
	s, repo := newTestServer(t)
	bundle := filepath.Join(repo, "knowledge")
	post(t, s, "/request/overview.md", url.Values{"text": {"fix"}}, nil)
	for name, args := range map[string][]string{
		"no actor":    {"-bundle", bundle, "-id", "r1", "-response", "done"},
		"bad actor":   {"-bundle", bundle, "-id", "r1", "-by", "someone", "-response", "done"},
		"no response": {"-bundle", bundle, "-id", "r1", "-by", "tool/1"},
		"unknown id":  {"-bundle", bundle, "-id", "r7", "-by", "tool/1", "-response", "done"},
	} {
		if err := ResolveRequest(&bytes.Buffer{}, args, s.now()); err == nil {
			t.Errorf("%s: resolve succeeded", name)
		}
	}
}

func TestBaselineConceptsAreReadOnly(t *testing.T) {
	s, repo := newTestServer(t)
	s.opts = Options{Baseline: "rules", Upstream: "https://example.com/kit"}
	file := filepath.Join(repo, "knowledge", "rules", "safe.md")
	before, _ := os.ReadFile(file)

	list := do(t, s, "GET", "/", nil).Body.String()
	for _, want := range []string{"0 of 1", `<span class="badge base">baseline</span>`} {
		if !strings.Contains(list, want) {
			t.Errorf("list with a baseline lacks %q", want)
		}
	}
	page := do(t, s, "GET", "/c/rules/safe.md", nil).Body.String()
	if strings.Contains(page, `action="/verify/`) || strings.Contains(page, `action="/request/`) || !strings.Contains(page, "https://example.com/kit") {
		t.Error("a baseline concept offers Verify or a request form, or does not point upstream")
	}
	if rec := do(t, s, "POST", "/verify/rules/safe.md", nil); rec.Code != http.StatusForbidden {
		t.Errorf("verifying a baseline concept: status %d", rec.Code)
	}
	if rec := post(t, s, "/request/rules/safe.md", url.Values{"text": {"change it"}}, nil); rec.Code != http.StatusForbidden {
		t.Errorf("requesting a change to a baseline concept: status %d", rec.Code)
	}
	if after, _ := os.ReadFile(file); string(after) != string(before) {
		t.Error("a baseline concept was changed")
	}
	// Verifying the only local concept finishes the review.
	if rec := do(t, s, "POST", "/verify/overview.md", nil); rec.Header().Get("Location") != "/" {
		t.Errorf("after the last local concept, redirected to %q", rec.Header().Get("Location"))
	}
}

func TestOverridesAreShownBothWays(t *testing.T) {
	s, repo := newTestServer(t)
	s.opts = Options{Baseline: "rules"}
	local := filepath.Join(repo, "knowledge", "local.md")
	text := "---\ntype: Rule\ntitle: Local safety\ndescription: Ours.\noverrides: rules/safe.md\n" +
		"generated: { by: tool/1, at: 2026-10-01T00:00:00Z }\n---\n\nOurs.\n"
	if err := os.WriteFile(local, []byte(text), 0o664); err != nil {
		t.Fatal(err)
	}
	if page := do(t, s, "GET", "/c/local.md", nil).Body.String(); !strings.Contains(page, `<dt>Overrides</dt><dd><a href="/c/rules/safe.md">`) {
		t.Error("the overriding concept does not link to what it overrides")
	}
	if page := do(t, s, "GET", "/c/rules/safe.md", nil).Body.String(); !strings.Contains(page, `<a href="/c/local.md">Local safety</a>`) {
		t.Error("the baseline concept does not say what overrides it")
	}
}
