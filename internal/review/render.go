package review

import (
	"html"
	"regexp"
	"strings"
)

// The renderer covers the markdown the knowledge bundle uses: headings,
// paragraphs, flat lists, tables, fenced code, and inline code, links, bold
// and footnotes. It is not a general markdown implementation.

var (
	heading   = regexp.MustCompile(`^(#{1,6}) +(.+)$`)
	bullet    = regexp.MustCompile(`^[*-] +(.*)$`)
	numbered  = regexp.MustCompile(`^\d+\. +(.*)$`)
	tableRule = regexp.MustCompile(`^\|[\s:|-]+\|$`)
	footDef   = regexp.MustCompile(`^\[\^([\w-]+)\]: *(.*)$`)
	linkRe    = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	boldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	footRef   = regexp.MustCompile(`\[\^([\w-]+)\]`)
)

// linkFunc turns a link target as written in a document into the href to
// use, and says whether it leaves the site.
type linkFunc func(target string) (href string, external bool)

// inline renders the inline markdown of one piece of text.
func inline(s string, link linkFunc) string {
	var b strings.Builder
	// Backticks split the text into prose and code; odd pieces are code.
	for i, part := range strings.Split(s, "`") {
		if i%2 == 1 {
			b.WriteString("<code>" + html.EscapeString(part) + "</code>")
			continue
		}
		part = html.EscapeString(part)
		part = linkRe.ReplaceAllStringFunc(part, func(m string) string {
			sub := linkRe.FindStringSubmatch(m)
			href, external := link(html.UnescapeString(sub[2]))
			attrs := ""
			if external {
				attrs = ` target="_blank" rel="noopener noreferrer"`
			}
			return `<a href="` + html.EscapeString(href) + `"` + attrs + `>` + sub[1] + `</a>`
		})
		part = boldRe.ReplaceAllString(part, "<strong>$1</strong>")
		part = footRef.ReplaceAllString(part, `<sup class="cite"><a href="#source-$1">$1</a></sup>`)
		b.WriteString(part)
	}
	return b.String()
}

// cells splits a table row into its trimmed cells.
func cells(row string) []string {
	row = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(row), "|"), "|")
	// A pipe inside a code span belongs to the cell.
	var out []string
	var cur strings.Builder
	inCode := false
	for _, r := range row {
		switch {
		case r == '`':
			inCode = !inCode
			cur.WriteRune(r)
		case r == '|' && !inCode:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	return append(out, strings.TrimSpace(cur.String()))
}

// render turns a concept body into HTML. Footnote definitions are dropped:
// the page lists the sources they name separately.
func render(body string, link linkFunc) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	var b strings.Builder
	in := func(s string) string { return inline(s, link) }

	for i := 0; i < len(lines); {
		line := lines[i]
		switch {
		case strings.TrimSpace(line) == "" || footDef.MatchString(line):
			i++

		case strings.HasPrefix(line, "```"):
			i++
			var code []string
			for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
				code = append(code, lines[i])
				i++
			}
			i++ // the closing fence
			b.WriteString("<pre><code>" + html.EscapeString(strings.Join(code, "\n")) + "</code></pre>\n")

		case heading.MatchString(line):
			m := heading.FindStringSubmatch(line)
			// The page title is the h1, so body headings start at h2.
			level := min(len(m[1])+1, 6)
			b.WriteString("<h" + string(rune('0'+level)) + ">" + in(m[2]) + "</h" + string(rune('0'+level)) + ">\n")
			i++

		case strings.HasPrefix(line, "|") && i+1 < len(lines) && tableRule.MatchString(strings.TrimSpace(lines[i+1])):
			b.WriteString("<table><thead><tr>")
			for _, c := range cells(line) {
				b.WriteString("<th>" + in(c) + "</th>")
			}
			b.WriteString("</tr></thead><tbody>\n")
			for i += 2; i < len(lines) && strings.HasPrefix(lines[i], "|"); i++ {
				b.WriteString("<tr>")
				for _, c := range cells(lines[i]) {
					b.WriteString("<td>" + in(c) + "</td>")
				}
				b.WriteString("</tr>\n")
			}
			b.WriteString("</tbody></table>\n")

		case bullet.MatchString(line) || numbered.MatchString(line):
			item, tag := bullet, "ul"
			if numbered.MatchString(line) {
				item, tag = numbered, "ol"
			}
			b.WriteString("<" + tag + ">\n")
			for i < len(lines) && item.MatchString(lines[i]) {
				text := item.FindStringSubmatch(lines[i])[1]
				// Indented lines continue the item.
				for i++; i < len(lines) && strings.HasPrefix(lines[i], " ") && strings.TrimSpace(lines[i]) != ""; i++ {
					text += " " + strings.TrimSpace(lines[i])
				}
				b.WriteString("<li>" + in(text) + "</li>\n")
			}
			b.WriteString("</" + tag + ">\n")

		default:
			var para []string
			for i < len(lines) && isParagraphLine(lines, i) {
				para = append(para, strings.TrimSpace(lines[i]))
				i++
			}
			b.WriteString("<p>" + in(strings.Join(para, " ")) + "</p>\n")
		}
	}
	return b.String()
}

// isParagraphLine reports whether line i continues a paragraph rather than
// starting another kind of block.
func isParagraphLine(lines []string, i int) bool {
	line := lines[i]
	return strings.TrimSpace(line) != "" &&
		!strings.HasPrefix(line, "```") &&
		!strings.HasPrefix(line, "|") &&
		!heading.MatchString(line) &&
		!bullet.MatchString(line) &&
		!numbered.MatchString(line) &&
		!footDef.MatchString(line)
}
