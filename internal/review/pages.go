package review

import (
	"html/template"
	"time"
)

var pages = template.Must(template.New("pages").Funcs(template.FuncMap{
	"when": func(t time.Time) string { return t.UTC().Format("2 Jan 2006, 15:04 UTC") },
}).Parse(pageTemplates))

const pageTemplates = `
{{define "head"}}<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.}} · knowledge review</title>
<style>
:root {
  --bg: #f6f7f9; --card: #ffffff; --text: #1c2330; --muted: #5f6b7c; --line: #dfe3ea;
  --accent: #2463eb; --accent-text: #ffffff; --ok: #16794a; --ok-bg: #e3f5ea;
  --warn: #9a5b00; --warn-bg: #fdf0d5; --code: #f0f2f6;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #12151b; --card: #1b2029; --text: #e6e9ef; --muted: #98a3b5; --line: #2c3442;
    --accent: #6ea0ff; --accent-text: #0b1220; --ok: #6fd6a0; --ok-bg: #173526;
    --warn: #f0c060; --warn-bg: #3a2e12; --code: #252c38;
  }
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--bg); color: var(--text);
  font: 16px/1.6 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; }
main { max-width: 880px; margin: 0 auto; padding: 24px 16px 140px; }
a { color: var(--accent); text-decoration: none; }
a:hover { text-decoration: underline; }
h1 { font-size: 1.7rem; line-height: 1.25; margin: 8px 0; }
h2 { font-size: 1.25rem; margin: 32px 0 8px; padding-bottom: 4px; border-bottom: 1px solid var(--line); }
h3 { font-size: 1.05rem; margin: 24px 0 8px; }
p, ul, ol { margin: 10px 0; }
li { margin: 4px 0; }
code { background: var(--code); border-radius: 4px; padding: 1px 5px;
  font: 0.9em ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
pre { background: var(--code); border-radius: 8px; padding: 12px 14px; overflow-x: auto; }
pre code { background: none; padding: 0; }
.scroll { overflow-x: auto; }
table { border-collapse: collapse; width: 100%; margin: 12px 0; font-size: 0.95rem; }
th, td { border: 1px solid var(--line); padding: 6px 10px; text-align: left; vertical-align: top; }
th { background: var(--code); }
.crumbs, .muted { color: var(--muted); font-size: 0.9rem; }
.lead { color: var(--muted); font-size: 1.05rem; margin: 4px 0 16px; }
.card { background: var(--card); border: 1px solid var(--line); border-radius: 10px; padding: 14px 18px; margin: 16px 0; }
.meta { display: grid; grid-template-columns: max-content 1fr; gap: 4px 16px; font-size: 0.92rem; margin: 0; }
.meta dt { color: var(--muted); }
.meta dd { margin: 0; overflow-wrap: anywhere; }
.badge { display: inline-block; border-radius: 999px; padding: 1px 10px; font-size: 0.8rem; font-weight: 600; white-space: nowrap; }
.badge.ok { color: var(--ok); background: var(--ok-bg); }
.badge.todo { color: var(--warn); background: var(--warn-bg); }
.chip { color: var(--muted); border: 1px solid var(--line); border-radius: 6px; padding: 0 6px; font-size: 0.78rem; white-space: nowrap; }
.rows { list-style: none; padding: 0; margin: 0; }
.rows li { display: grid; grid-template-columns: 150px 1fr; gap: 12px; padding: 10px 0; border-top: 1px solid var(--line); margin: 0; }
.rows li:first-child { border-top: 0; }
.rows .desc { color: var(--muted); font-size: 0.92rem; }
.bar { height: 10px; background: var(--line); border-radius: 999px; overflow: hidden; margin: 10px 0; }
.bar div { height: 100%; background: var(--ok); }
.cite a { font-size: 0.75rem; }
.actions { position: fixed; left: 0; right: 0; bottom: 0; background: var(--card); border-top: 1px solid var(--line); }
.actions .inner { max-width: 880px; margin: 0 auto; padding: 12px 16px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.actions .say { flex: 1 1 260px; font-size: 0.9rem; color: var(--muted); }
button, .button { font: inherit; font-weight: 600; border-radius: 8px; padding: 8px 18px; cursor: pointer;
  border: 1px solid var(--line); background: var(--card); color: var(--text); display: inline-block; }
button.primary, .button.primary { background: var(--accent); color: var(--accent-text); border-color: var(--accent); }
.button:hover { text-decoration: none; }
form { margin: 0; }
.badge.wait { color: var(--accent); background: var(--code); }
.badge.base { color: var(--muted); background: var(--code); }
textarea { width: 100%; min-height: 96px; font: inherit; color: var(--text); background: var(--bg);
  border: 1px solid var(--line); border-radius: 8px; padding: 8px 10px; resize: vertical; }
.request { border-top: 1px solid var(--line); padding: 10px 0; }
.request:first-of-type { border-top: 0; }
.request .text { white-space: pre-wrap; overflow-wrap: anywhere; margin: 4px 0; }
.request .answer { border-left: 3px solid var(--ok); padding-left: 10px; margin: 8px 0 0; }
.row { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; margin-top: 8px; }
.source { font: 0.85rem/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; border-collapse: collapse; width: auto; }
.source td { border: 0; padding: 0 10px; white-space: pre; }
.source td.n { color: var(--muted); text-align: right; user-select: none; border-right: 1px solid var(--line); }
.source tr:target td { background: var(--warn-bg); }
</style>
</head>
<body>
<main>
{{end}}

{{define "badge"}}{{if eq . "verified"}}<span class="badge ok">{{.}}</span>{{else if eq . "change requested"}}<span class="badge wait">{{.}}</span>{{else if eq . "baseline"}}<span class="badge base">{{.}}</span>{{else}}<span class="badge todo">{{.}}</span>{{end}}{{end}}

{{define "requests"}}{{range .}}<div class="request">
  <div class="muted"><code>{{.ID}}</code> · {{if .Open}}<span class="badge wait">waiting for an agent</span>{{else}}<span class="badge ok">done</span>{{end}} · requested by <code>{{.By}}</code> on {{when .At}}</div>
  <div class="text">{{.Text}}</div>
  {{if .Open}}<form method="post" action="/withdraw/{{.ID}}"><button type="submit">Withdraw</button></form>
  {{else}}<div class="answer"><div class="muted">Answered by <code>{{.ResolvedBy}}</code>{{with .ResolvedAt}} on {{when .}}{{end}}</div><div class="text">{{.Response}}</div></div>{{end}}
</div>{{end}}{{end}}

{{define "list"}}{{template "head" "Concepts"}}
<p class="crumbs">Knowledge review · verifying as <code>{{.Actor}}</code></p>
<h1>Concepts to review</h1>
<div class="card">
  <strong>{{.Verified}} of {{.Total}}</strong> of this repository's concepts verified by a person
  <div class="bar"><div style="width: {{.Percent}}%"></div></div>
  {{if .Waiting}}<p class="muted">{{.Waiting}} change request{{if ne .Waiting 1}}s{{end}} waiting for an agent. Ask one to process the change requests, then come back.</p>{{end}}
  {{if .Next}}<a class="button primary" href="/c/{{.Next}}">Review the next concept</a>
  {{else if eq .Verified .Total}}<span class="badge ok">Everything is verified</span>
  <p class="muted">Commit the changed files under the knowledge folder to keep the verifications.</p>
  {{else}}<span class="badge wait">Nothing left for you until the change requests are processed</span>{{end}}
</div>
{{range .Sections}}
<h2 id="{{.Name}}">{{if .Name}}{{.Name}}{{else}}start here{{end}}</h2>
<ul class="rows">
{{range .Rows}}<li>
  <div>{{template "badge" .State}}</div>
  <div><a href="/c/{{.Path}}"><strong>{{.Title}}</strong></a> <span class="chip">{{.Type}}</span>
  <div class="desc">{{.Description}}</div></div>
</li>{{end}}
</ul>
{{end}}
<h2 id="requests">requests about the whole bundle</h2>
<div class="card">
  {{template "requests" .Requests}}
  <form method="post" action="/request/">
    <p class="muted">Something missing, or wrong in more than one place? Describe it and an agent will update the bundle.</p>
    <textarea name="text" maxlength="{{.MaxRequest}}" required placeholder="For example: add a concept about how releases are made."></textarea>
    <div class="row"><button type="submit">Request a change</button></div>
  </form>
</div>
</main>
</body>
</html>{{end}}

{{define "concept"}}{{template "head" .Title}}
<p class="crumbs"><a href="/">All concepts</a> · {{.Path}} · {{.Position}} of {{.Total}}</p>
<h1>{{.Title}}</h1>
<p class="lead">{{.Description}}</p>
<p>
  {{template "badge" .State}}
  <span class="chip">{{.Type}}</span>
  {{if .Status}}<span class="chip">{{.Status}}</span>{{end}}
</p>
<div class="card">
<dl class="meta">
  {{if .Generated.By}}<dt>Written by</dt><dd><code>{{.Generated.By}}</code> on {{when .Generated.At}}</dd>{{end}}
  <dt>Verified by</dt><dd>{{range .Verified}}<div><code>{{.By}}</code> on {{when .At}}</div>{{else}}nobody yet{{end}}</dd>
  {{if .Tags}}<dt>Tags</dt><dd>{{.Tags}}</dd>{{end}}
  {{with .Resource}}<dt>Describes</dt><dd><a href="{{.Href}}"{{if .External}} target="_blank" rel="noopener noreferrer"{{end}}>{{.Title}}</a></dd>{{end}}
  {{with .Overrides}}<dt>Overrides</dt><dd><a href="{{.Href}}">{{.Title}}</a> (this concept replaces that baseline concept here)</dd>{{end}}
  {{if .OverriddenBy}}<dt>Overridden by</dt><dd>{{range .OverriddenBy}}<div><a href="{{.Href}}">{{.Title}}</a></div>{{end}}</dd>{{end}}
  {{if .Sources}}<dt>Sources</dt><dd>{{range .Sources}}<div id="source-{{.ID}}"><span class="chip">{{.ID}}</span>
    <a href="{{.Href}}"{{if .External}} target="_blank" rel="noopener noreferrer"{{end}}>{{.Title}}</a></div>{{end}}</dd>{{end}}
</dl>
</div>
<article class="scroll">
{{.Body}}
</article>
{{if not .Managed}}<h2 id="requests">Change requests</h2>
<div class="card">
  {{template "requests" .Requests}}
  <form method="post" action="/request/{{.Path}}">
    <p class="muted">Not right? Describe what should change. An agent updates the concept and you review it again.</p>
    <textarea name="text" maxlength="{{.MaxRequest}}" required placeholder="For example: the section on error handling is out of date, check the retry code."></textarea>
    <div class="row"><button type="submit">Request a change</button></div>
  </form>
</div>{{end}}
</main>
<div class="actions"><div class="inner">
{{if .Managed}}
  <div class="say">This concept is part of the baseline that comes with the kit. It is reviewed and changed upstream{{if .Upstream}}, at <a href="{{.Upstream}}" target="_blank" rel="noopener noreferrer">{{.Upstream}}</a>{{end}}. To differ in this repository, add a local concept that overrides it.</div>
  {{if .Next}}<a class="button primary" href="/c/{{.Next}}">Next concept</a>{{else}}<a class="button primary" href="/">All concepts</a>{{end}}
{{else if .Requested}}
  <div class="say">A change was requested. This concept comes back for review once an agent has updated it.</div>
  {{if .Next}}<a class="button primary" href="/c/{{.Next}}">Next concept</a>{{else}}<a class="button primary" href="/">All concepts</a>{{end}}
{{else if .Verifiable}}
  <div class="say">{{if eq .State "updated, review again"}}An agent updated this concept after your request; its answer is under Change requests. {{else if .HasHuman}}This concept was rewritten after {{.Human.By}} verified it. {{end}}Verifying records that you read this against its sources and it is correct.</div>
  {{if .Next}}<a class="button" href="/c/{{.Next}}">Skip</a>{{end}}
  <form method="post" action="/verify/{{.Path}}"><button class="primary" type="submit">Verify as {{.Actor}}</button></form>
{{else}}
  <div class="say">Verified by <code>{{.Human.By}}</code> on {{when .Human.At}}.</div>
  {{if .Mine}}<form method="post" action="/unverify/{{.Path}}"><button type="submit">Remove my verification</button></form>{{end}}
  {{if .Next}}<a class="button primary" href="/c/{{.Next}}">Next concept</a>{{else}}<a class="button primary" href="/">All concepts</a>{{end}}
{{end}}
</div></div>
</body>
</html>{{end}}

{{define "source"}}{{template "head" .Path}}
<p class="crumbs"><a href="/">All concepts</a> · source file, read-only</p>
<h1><code>{{.Path}}</code></h1>
{{if .Note}}<p class="muted">{{.Note}}</p>{{end}}
<div class="card scroll">
<table class="source">
{{range .Lines}}<tr id="L{{.N}}"><td class="n">{{.N}}</td><td>{{.Text}}</td></tr>
{{end}}</table>
</div>
</main>
</body>
</html>{{end}}

{{define "folder"}}{{template "head" .Path}}
<p class="crumbs"><a href="/">All concepts</a> · source folder, read-only</p>
<h1><code>{{.Path}}</code></h1>
<div class="card">
<ul>{{range .Files}}<li><a href="/src/{{.}}">{{.}}</a></li>{{end}}</ul>
</div>
</main>
</body>
</html>{{end}}
`
