package official

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/width"
)

// Reading tells how to read the text layer of the papers of one source.
type Reading struct {
	// Halves is true when a landscape page holds two pages side by side, such as the two days of a final. Each half is read alone, the left half first.
	Halves bool
	// Raw reads the text in the order of the PDF, and not in the order of the layout. It is for papers with several columns of problems, which the layout puts on the same lines.
	Raw bool
	// End matches the line where the problems of a page end, such as the start of the notes for the contestants.
	End *regexp.Regexp
	// Cover matches the instructions of the cover, up to their end. When the paper has more than one page, a first page that matches Cover is the cover, and it is dropped. When the paper has one page, the text of the page up to the end of the match is dropped.
	Cover *regexp.Regexp
	// Japanese is true for papers in Japanese. Their text is readable when it has enough letters, because most of its letters are not ASCII letters.
	Japanese bool
}

// Readings are the rules to read the papers of the sources that need them. The other sources use Text.
var Readings = map[string]Reading{
	"om-sem":           {Halves: true, End: regexp.MustCompile(`(?m)^\s*Informacje dla uczestnika`)},
	"bundeswettbewerb": {Raw: true},
	"imojp":            {Japanese: true},
	"ukmt":             {Halves: true, End: regexp.MustCompile(`(?m)^\s*© [0-9]{4} UK Mathematics Trust`), Cover: regexp.MustCompile(`(?s)\A.*Do not turn over until[^\n]*`)},
}

// Sections are the readers of the sources whose papers are sections of an HTML page. A reader gives the text of the section with the id that is the fragment of the URL of the paper, or "" when the page has no such section.
var Sections = map[string]func(page, id string) string{
	"pamo-official": PAMOText,
}

// Parts are the readers of the sources whose files hold more than one paper of the layout. A reader gives the part of the text of a file that is the paper with the id paper, or "" when the file has no such part.
var Parts = map[string]func(text, paper string) string{
	"tuymaada-official": TuymaadaPart,
	"spm-opm":           OPMPart,
}

// Read gives the text of a paper of a source, with a form feed between pages. It uses the rules of Readings for the source. A file that is not a PDF, such as an HTML page that shows the scans of a paper, gives ErrNoText.
func Read(ctx context.Context, source string, pdf []byte) (string, error) {
	if !bytes.Contains(pdf[:min(len(pdf), 1024)], []byte("%PDF-")) {
		return "", ErrNoText
	}
	r, ok := Readings[source]
	if !ok {
		return Text(ctx, pdf)
	}
	var pages []string
	if w, h, err := pageSize(ctx, pdf); r.Halves && err == nil && w > h {
		left, err := pdftotext(ctx, pdf, "-layout", "-x", "0", "-y", "0", "-W", strconv.Itoa(w/2+1), "-H", strconv.Itoa(h+1))
		if err != nil {
			return "", err
		}
		right, err := pdftotext(ctx, pdf, "-layout", "-x", strconv.Itoa(w/2), "-y", "0", "-W", strconv.Itoa(w/2+1), "-H", strconv.Itoa(h+1))
		if err != nil {
			return "", err
		}
		ls, rs := strings.Split(left, "\f"), strings.Split(right, "\f")
		n := min(len(ls), len(rs))
		ls, rs = r.cover(ls[:n]), r.cover(rs[:n])
		for i := range min(len(ls), len(rs)) {
			pages = append(pages, r.cut(ls[i])+"\n"+r.cut(rs[i]))
		}
	} else {
		args := []string{"-layout"}
		if r.Raw {
			args = nil
		}
		text, err := pdftotext(ctx, pdf, args...)
		if err != nil {
			return "", err
		}
		for _, p := range r.cover(strings.Split(text, "\f")) {
			pages = append(pages, r.cut(p))
		}
	}
	text := strings.Join(pages, "\f")
	if r.Japanese && !hasLetters(text) || !r.Japanese && !readable(text) {
		return "", ErrNoText
	}
	return text, nil
}

// leastLetters is the smallest number of letters of a readable paper in Japanese. A scan with no text layer gives none.
const leastLetters = 100

// hasLetters tells if a text has at least leastLetters letters.
func hasLetters(text string) bool {
	n := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n >= leastLetters
}

// cut removes the text of a page after the line that End matches.
func (r Reading) cut(page string) string {
	if r.End == nil {
		return page
	}
	if m := r.End.FindStringIndex(page); m != nil {
		return page[:m[0]]
	}
	return page
}

// cover drops the cover of a paper from its pages, see Cover. pdftotext ends the text with a form feed, so the last page can be empty, and it does not count.
func (r Reading) cover(pages []string) []string {
	if r.Cover == nil || len(pages) == 0 {
		return pages
	}
	n := len(pages)
	if n > 1 && strings.TrimSpace(pages[n-1]) == "" {
		n--
	}
	m := r.Cover.FindStringIndex(pages[0])
	switch {
	case m == nil:
		return pages
	case n > 1:
		return append([]string{""}, pages[1:]...)
	}
	out := append([]string{}, pages...)
	out[0] = pages[0][m[1]:]
	return out
}

// pdftotext runs pdftotext with the arguments on the bytes of a PDF.
func pdftotext(ctx context.Context, pdf []byte, args ...string) (string, error) {
	args = append(args, "-enc", "UTF-8", "-", "-")
	cmd := exec.CommandContext(ctx, "pdftotext", args...) // #nosec G204 -- the arguments are fixed flags and numbers.
	cmd.Stdin = bytes.NewReader(pdf)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

var (
	pageSizeRE = regexp.MustCompile(`(?m)^Page size:\s+([0-9.]+) x ([0-9.]+)`)
	pageRotRE  = regexp.MustCompile(`(?m)^Page rot:\s+([0-9]+)`)
)

// pageSize gives the width and the height of the first page of a PDF, in points, as the page shows. A page that turns by 90 or 270 degrees swaps its width and height.
func pageSize(ctx context.Context, pdf []byte) (w, h int, err error) {
	cmd := exec.CommandContext(ctx, "pdfinfo", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("pdfinfo: %w", err)
	}
	w, h, ok := infoSize(out)
	if !ok {
		return 0, 0, fmt.Errorf("pdfinfo gives no page size")
	}
	return w, h, nil
}

// infoSize reads the page size from the output of pdfinfo, with the width and height swapped for a page that turns by 90 or 270 degrees.
func infoSize(out []byte) (w, h int, ok bool) {
	m := pageSizeRE.FindSubmatch(out)
	if m == nil {
		return 0, 0, false
	}
	fw, _ := strconv.ParseFloat(string(m[1]), 64)
	fh, _ := strconv.ParseFloat(string(m[2]), 64)
	if r := pageRotRE.FindSubmatch(out); r != nil && (string(r[1]) == "90" || string(r[1]) == "270") {
		fw, fh = fh, fw
	}
	return int(fw), int(fh), true
}

// readable tells if at least half of the letters of a text are ASCII letters.
func readable(text string) bool {
	var ascii, all int
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsSymbol(r) {
			all++
			if r < unicode.MaxASCII {
				ascii++
			}
		}
	}
	return ascii*2 >= all
}

// English tells if the code of a language is the code of English. The papers of imo-official.org use three letters.
func English(code string) bool { return code == "" || code == "en" || code == "eng" }

// stopLetters are the words of one letter of a language. In the text of a paper they are words, not names of numbers or points.
var stopLetters = map[string]string{"pl": "aiouwz", "pol": "aiouwz"}

// mathRE finds the formulas of a TeX text: $...$, $$...$$, \(...\) and \[...\].
var mathRE = regexp.MustCompile(`(?s)\$\$(.*?)\$\$|\$(.*?)\$|\\\((.*?)\\\)|\\\[(.*?)\\\]`)

// commandRE matches a TeX command, such as \frac.
var commandRE = regexp.MustCompile(`\\[A-Za-z]+`)

// PaperFormulas gives the formula tokens of the text of a paper in the language lang: each number, each letter alone, such as the name of a number or a point, and each run of two to five capital ASCII letters, such as ABC. A word of one letter of the language is not a token. The text of a paper does not mark its formulas, so the words of the text give no tokens, except these. A run of Japanese letters is a word, so it is not a token, and the full-width forms of letters and digits count as the ASCII forms.
func PaperFormulas(text, lang string) map[string]bool {
	stop := stopLetters[lang]
	out := map[string]bool{}
	for _, t := range runs(width.Fold.String(text)) {
		switch {
		case japanese(t[0]):
		case unicode.IsDigit(t[0]):
			out[number(string(t))] = true
		case len(t) == 1:
			if !strings.ContainsRune(stop, t[0]) {
				out[string(t)] = true
			}
		case len(t) <= 5 && capitals(t):
			out[string(t)] = true
		}
	}
	return out
}

// TeXFormulas gives the formula tokens of a TeX text: each number of the text, and the letters and the runs of capital letters of its formulas, as PaperFormulas gives them. A TeX command is not a token.
func TeXFormulas(text string) map[string]bool {
	out := map[string]bool{}
	for _, t := range runs(text) {
		if unicode.IsDigit(t[0]) {
			out[number(string(t))] = true
		}
	}
	for _, m := range mathRE.FindAllStringSubmatch(text, -1) {
		f := commandRE.ReplaceAllString(m[1]+" "+m[2]+" "+m[3]+" "+m[4], " ")
		for _, t := range runs(f) {
			if !unicode.IsDigit(t[0]) && (len(t) == 1 || (len(t) <= 5 && capitals(t))) {
				out[string(t)] = true
			}
		}
	}
	return out
}

// runs cuts a text into runs of letters and runs of digits. A run of Japanese letters is apart from the letters of other scripts, as in "nを" of a Japanese paper.
func runs(s string) [][]rune {
	var out [][]rune
	var cur []rune
	digit, ja := false, false
	for _, r := range s {
		l, d := unicode.IsLetter(r), unicode.IsDigit(r)
		if (!l && !d) || (len(cur) > 0 && (d != digit || (l && !digit && japanese(r) != ja))) {
			if len(cur) > 0 {
				out = append(out, cur)
			}
			cur = nil
		}
		if l || d {
			cur, digit, ja = append(cur, r), d, l && japanese(r)
		}
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

// japanese tells if a letter is a kanji, a hiragana or a katakana.
func japanese(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
}

func capitals(t []rune) bool {
	for _, r := range t {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// number removes the zeros at the start of a number.
func number(s string) string {
	if t := strings.TrimLeft(s, "0"); t != "" {
		return t
	}
	return "0"
}

// FormulaSignal gives the formula signal of a TeX text and the text of a paper in the language lang: the Jaccard index of their formula tokens. It compares two texts in two languages.
func FormulaSignal(text, paper, lang string) float64 {
	a, b := TeXFormulas(text), PaperFormulas(paper, lang)
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := 0
	for t := range a {
		if b[t] {
			n++
		}
	}
	return float64(n) / float64(len(a)+len(b)-n)
}
