package official

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

var (
	// totCell is one cell of a TeX paper of the Tournament: a label, as "1.", "$1$." or "а)", and its text. A cell with no label holds the author or a note.
	totCell = regexp.MustCompile(`(?s)\\begin\{tabular\}\{p\{[^}]*\}p\{[^}]*\}\}(.*?)&(.*?)\\end\{tabular\}`)
	// totTeXNum is the label of a problem.
	totTeXNum = regexp.MustCompile(`^\$?(\d{1,2})\$?\.$`)
	// totTeXPart is the label of a part.
	totTeXPart = regexp.MustCompile(`^[а-ж]\)$`)
	// totTeXComment is a comment of TeX, to the end of the line.
	totTeXComment = regexp.MustCompile(`(?m)(^|[^\\])%.*$`)
	// totTeXGroup is a group that only sets the font, as {\sl text}.
	totTeXGroup = regexp.MustCompile(`\{\\(?:sl|bf|it|em|rm)\s+([^{}]*)\}`)
	// totTeXSpace is a command that only makes space or a break.
	totTeXSpace = regexp.MustCompile(`\\(?:hfill|noindent|qquad|quad|break|vfill|newline|smallskip|medskip|bigskip)\b|\\\\|~`)
	// totTeXPunc is a mark of the sentence at the end of a formula, as $x<2.$
	totTeXPunc = regexp.MustCompile(`([.,;:])\$`)
	// totTeXCommand is a command of TeX.
	totTeXCommand = regexp.MustCompile(`\\[a-zA-Z]+`)
)

// TotTeXDecode gives the text of a TeX file of the Tournament, which is in UTF-8, CP866 or Windows-1251. The small letters а to п are 0xA0 to 0xAF in CP866, where Windows-1251 has few letters.
func TotTeXDecode(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	dos, win := 0, 0
	for _, c := range b {
		switch {
		case c >= 0xA0 && c <= 0xAF:
			dos++
		case c >= 0xF0:
			win++
		}
	}
	dec := charmap.Windows1251.NewDecoder()
	if dos > win {
		dec = charmap.CodePage866.NewDecoder()
	}
	s, err := dec.String(string(b))
	if err != nil {
		return ""
	}
	return s
}

// TotTeX reads the problems of a TeX file of papers of the Tournament of Towns, as os30os.tex. Each paper has a heading, as "8 -- 9 классы, сложный вариант", and each problem, part and author is a cell of a tabular. The positions are ja1, jo1, sa1 or so1 (spec 04).
func TotTeX(src, edition, url string) []RuProblem {
	src = totTeXComment.ReplaceAllString(src, "$1")
	var out []RuProblem
	seen := map[string]bool{}
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		group, tier, ok := TotHeading(lines[i])
		if !ok || tier == 0 || strings.Contains(lines[i], "Итог") {
			continue
		}
		j := i + 1
		for j < len(lines) {
			if _, _, ok := TotHeading(lines[j]); ok || strings.Contains(lines[j], "ТУРНИР ГОРОДОВ") {
				break
			}
			j++
		}
		prefix := string([]byte{group, tier})
		body := strings.Join(lines[i+1:j], "\n")
		i = j - 1
		if seen[prefix] {
			continue
		}
		seen[prefix] = true
		out = append(out, totTeXPaper(body, prefix, edition, url)...)
	}
	return out
}

// totTeXPaper reads the problems of one paper from its cells.
func totTeXPaper(body, prefix, edition, url string) []RuProblem {
	var out []RuProblem
	var cur *strings.Builder
	var nums []int
	for _, m := range totCell.FindAllStringSubmatch(body, -1) {
		label := strings.TrimSpace(totTeXSpace.ReplaceAllString(m[1], ""))
		text := strings.Join(strings.Fields(m[2]), " ")
		switch {
		case totTeXNum.MatchString(label):
			n, _ := strconv.Atoi(totTeXNum.FindStringSubmatch(label)[1])
			nums = append(nums, n)
			cur = &strings.Builder{}
			out = append(out, RuProblem{Edition: edition, Position: prefix + strconv.Itoa(n), URL: url, Tool: "tex"})
			cur.WriteString(text)
		case cur == nil:
		case totTeXPart.MatchString(label):
			cur.WriteString("\n" + label + " " + text)
		case label == "" && strings.Contains(text, `\hfill`):
			// The author, set to the right.
		case strings.Contains(cur.String(), "\n"):
			// A note after the parts is about the whole problem.
			cur.WriteString("\n" + strings.TrimSpace(label+" "+text))
		default:
			cur.WriteString(" " + label + " " + text)
		}
		if cur != nil {
			out[len(out)-1].Text = cur.String()
		}
	}
	// The numbers go 1, 2, 3 in a paper; a gap means a cell this reader did not see.
	for k, n := range nums {
		if n != k+1 {
			return nil
		}
	}
	for k := range out {
		out[k] = totTeXClean(out[k])
	}
	return out
}

// totTeXClean makes the text of a problem from its TeX: it drops the commands of layout and keeps the math. A command left out of the math sends the problem to review.
func totTeXClean(p RuProblem) RuProblem {
	s := p.Text
	s = strings.NewReplacer(`\"е`, "ё", `\"Е`, "Ё", `\"{е}`, "ё", `\-`, "").Replace(s)
	s = totTeXSpace.ReplaceAllString(s, " ")
	s = totTeXUnbox(s)
	s = totTeXGroup.ReplaceAllString(s, "$1")
	s = totTeXPunc.ReplaceAllString(s, "$$$1")
	parts := strings.Split(s, "$")
	layout := len(parts)%2 == 0
	for k := 0; k < len(parts); k += 2 {
		parts[k] = strings.ReplaceAll(parts[k], `\dots`, "...")
		parts[k] = totDash.ReplaceAllString(parts[k], " — ")
		if totTeXCommand.MatchString(parts[k]) || strings.ContainsAny(parts[k], `{}\`) {
			layout = true
		}
	}
	s = strings.Join(parts, "$")
	lines := strings.Split(s, "\n")
	for k, l := range lines {
		lines[k] = strings.Join(strings.Fields(l), " ")
	}
	s = strings.Join(lines, "\n")
	if totFigure.MatchString(s) || strings.Contains(s, `\begin{picture}`) || strings.Contains(s, `\includegraphics`) {
		p.Figure = true
	}
	if layout || totVerbal.MatchString(s) {
		p.Flags = append(p.Flags, "layout")
	}
	p.Text = strings.TrimSpace(s)
	return p
}

// totTeXUnbox drops the boxes around text or math, as \hbox{$a_n=a_{n-1}+1$,}, and keeps what they hold.
func totTeXUnbox(s string) string {
	for _, box := range []string{`\hbox{`, `\mbox{`} {
		for {
			i := strings.Index(s, box)
			if i < 0 {
				break
			}
			depth, j := 1, i+len(box)
			for ; j < len(s) && depth > 0; j++ {
				switch s[j] {
				case '{':
					depth++
				case '}':
					depth--
				}
			}
			if depth > 0 {
				return s
			}
			s = s[:i] + s[i+len(box):j-1] + s[j:]
		}
	}
	return s
}
