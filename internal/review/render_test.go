package review

import (
	"strings"
	"testing"
)

func testLink(target string) (string, bool) {
	if strings.HasPrefix(target, "https://") {
		return target, true
	}
	return "/c/" + target, false
}

func TestInline(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain text is escaped", `a < b & "c"`, `a &lt; b &amp; &#34;c&#34;`},
		{"code", "run `go test ./...` now", "run <code>go test ./...</code> now"},
		{"code is escaped and not formatted", "`<b> **x** [a](b)`", "<code>&lt;b&gt; **x** [a](b)</code>"},
		{"bold", "a **b** c", "a <strong>b</strong> c"},
		{"local link", "see [app](app.md)", `see <a href="/c/app.md">app</a>`},
		{"external link", "[spec](https://example.com/a?b=1&c=2)", `<a href="https://example.com/a?b=1&amp;c=2" target="_blank" rel="noopener noreferrer">spec</a>`},
		{"footnote", "true.[^src-1]", `true.<sup class="cite"><a href="#source-src-1">src-1</a></sup>`},
		{"script is inert", "<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;"},
	}
	for _, tt := range tests {
		if got := inline(tt.in, testLink); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, got, tt.want)
		}
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"heading is one level down", "# Rule\n", "<h2>Rule</h2>\n"},
		{"paragraph lines are joined", "one\ntwo\n\nthree\n", "<p>one two</p>\n<p>three</p>\n"},
		{"bullets with a continuation", "* one\n  more\n* two\n", "<ul>\n<li>one more</li>\n<li>two</li>\n</ul>\n"},
		{"numbered", "1. one\n2. two\n   more\n", "<ol>\n<li>one</li>\n<li>two more</li>\n</ol>\n"},
		{"fenced code is verbatim", "```\na < b\n# not a heading\n```\n", "<pre><code>a &lt; b\n# not a heading</code></pre>\n"},
		{"table", "| A | B |\n|---|---|\n| `x|y` | z |\n",
			"<table><thead><tr><th>A</th><th>B</th></tr></thead><tbody>\n<tr><td><code>x|y</code></td><td>z</td></tr>\n</tbody></table>\n"},
		{"footnote definitions are dropped", "text\n\n[^a]: A source\n", "<p>text</p>\n"},
		{"paragraph ends at a list", "intro:\n* one\n", "<p>intro:</p>\n<ul>\n<li>one</li>\n</ul>\n"},
	}
	for _, tt := range tests {
		if got := render(tt.in, testLink); got != tt.want {
			t.Errorf("%s:\n got %q\nwant %q", tt.name, got, tt.want)
		}
	}
}
