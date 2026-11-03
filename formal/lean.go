// Package formal imports the formal statements and proofs of the open Lean sources into the content repository, and reads the results of their checks (spec 08).
//
// The import copies the Lean files of a source to lean/vendor/<source>/<commit>/ in the content, maps each statement to its problem by competition, year and position, and writes one formal record for each statement and one proof record for each proof. The checks run on a Linux host with the pinned toolchain. Their results come back as files that Record reads.
package formal

import (
	"regexp"
	"strings"
)

// Decl is one declaration in a Lean file.
type Decl struct {
	// Keyword is theorem, lemma, problem, abbrev, def or determine.
	Keyword string
	// Name is the full name, with the namespaces that are open at the declaration.
	Name string
	// Short is the name as the file writes it.
	Short string
	// Text is the declaration without its proof or value, from the keyword to the := at depth 0. For a definition the value is kept.
	Text string
	// Body is the text after the :=, up to the next declaration.
	Body string
	// Line is the line of the keyword, from 1.
	Line int
}

var declRE = regexp.MustCompile(`(?m)^((?:@\[[^\]]*\]\s*)?(?:(?:private|protected|noncomputable|public)\s+)*)(theorem|lemma|problem|abbrev|def|determine)\s+([^\s(:{\[]+)`)

// nsRE also reads the modifiers of a section, as in noncomputable section and @[expose] public section, so that its end does not close a namespace.
var nsRE = regexp.MustCompile(`(?m)^(?:@\[[^\]]*\]\s*)?(?:(?:noncomputable|public|private)\s+)*(namespace|end|section)\b[ \t]*([^\s]*)`)

// Decls returns the top-level declarations of a Lean file, in order. It reads the namespaces to give full names. It is a light reader for well-formatted files, not a Lean parser.
func Decls(src string) []Decl {
	src = stripComments(src)
	type mark struct {
		pos  int
		kind string
		name string
	}
	var marks []mark
	for _, m := range nsRE.FindAllStringSubmatchIndex(src, -1) {
		marks = append(marks, mark{pos: m[0], kind: src[m[2]:m[3]], name: src[m[4]:m[5]]})
	}
	locs := declRE.FindAllStringSubmatchIndex(src, -1)
	var out []Decl
	var stack []mark
	mi := 0
	for i, m := range locs {
		for mi < len(marks) && marks[mi].pos < m[0] {
			k := marks[mi]
			switch k.kind {
			case "namespace", "section":
				stack = append(stack, k)
			case "end":
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
			mi++
		}
		end := len(src)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		for _, k := range marks {
			if k.pos > m[0] && k.pos < end {
				end = k.pos
				break
			}
		}
		chunk := src[m[0]:end]
		short := src[m[6]:m[7]]
		var ns []string
		for _, k := range stack {
			if k.kind == "namespace" {
				ns = append(ns, k.name)
			}
		}
		name := short
		if len(ns) > 0 && !strings.HasPrefix(short, "_root_.") {
			name = strings.Join(ns, ".") + "." + short
		}
		name = strings.TrimPrefix(name, "_root_.")
		kw := src[m[4]:m[5]]
		text, body := chunk, ""
		if at := topAssign(chunk); at >= 0 {
			text, body = chunk[:at], chunk[at+2:]
		}
		text = strings.TrimSpace(text)
		if kw == "abbrev" || kw == "def" || kw == "determine" {
			text = strings.TrimSpace(chunk)
		}
		out = append(out, Decl{Keyword: kw, Name: name, Short: short, Text: text, Body: body, Line: strings.Count(src[:m[0]], "\n") + 1})
	}
	return out
}

// topAssign returns the index of the first := outside brackets, or -1.
func topAssign(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 && i+1 < len(s) && s[i+1] == '=' {
				return i
			}
		}
	}
	return -1
}

// stripComments blanks the comments of a Lean file and keeps its lines, so that line numbers stay the same.
func stripComments(s string) string {
	b := []byte(s)
	depth := 0
	for i := 0; i < len(b); i++ {
		switch {
		case i+1 < len(b) && b[i] == '/' && b[i+1] == '-':
			depth++
			b[i], b[i+1] = ' ', ' '
			i++
		case depth > 0 && i+1 < len(b) && b[i] == '-' && b[i+1] == '/':
			depth--
			b[i], b[i+1] = ' ', ' '
			i++
		case depth > 0:
			if b[i] != '\n' {
				b[i] = ' '
			}
		case i+1 < len(b) && b[i] == '-' && b[i+1] == '-':
			for i < len(b) && b[i] != '\n' {
				b[i] = ' '
				i++
			}
		}
	}
	return string(b)
}

var sorryRE = regexp.MustCompile(`\bsorry\b`)

// HasSorry reports whether a Lean text uses sorry outside comments.
func HasSorry(s string) bool { return sorryRE.MatchString(stripComments(s)) }

// Standalone rewrites the Compfiles commands of a statement to plain Lean, so that the text compiles with Mathlib alone: problem becomes theorem and determine becomes abbrev.
func Standalone(text string) string {
	text = regexp.MustCompile(`(?m)^problem\b`).ReplaceAllString(text, "theorem")
	return regexp.MustCompile(`(?m)^determine\b`).ReplaceAllString(text, "abbrev")
}
