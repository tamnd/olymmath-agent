package official

import (
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"github.com/tamnd/olymmath-agent/figure"
)

// ruSmall marks a word of a PDF that is much smaller than the words of its page, as an exponent or an index. The text layer gives such a word apart from its base, so a problem that holds one is flagged.
const ruSmall = "\x03"

// ruWordRE is a word of the output of pdftotext -raw -bbox, and ruPageRE the start of a page.
var (
	ruWordRE = regexp.MustCompile(`<word xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)" yMax="([\d.]+)">([^<]*)</word>`)
	ruPageRE = regexp.MustCompile(`<page\b`)
)

// RuPDFLine is one text line of a PDF and its page, from 1.
type RuPDFLine struct {
	Text string
	Page int
}

// RuPDFLines gives the text lines of a PDF from the output of pdftotext -raw -bbox, whose words come in the order of the content of each page. A line ends where the next word goes back to the left or down by more than a line. Two words that touch are joined with no space, as "9" and ".". A word below 0.8 of the median height of its page, as an exponent or an index, is marked with ruSmall. An old TeX font with no Unicode table gives its Cyrillic letters as the Latin-1 letters of the KOI8-R codes; such a page is put back to Cyrillic, and koi8 tells that a page was.
func RuPDFLines(b []byte) (lines []RuPDFLine, koi8 bool) {
	s := string(b)
	starts := ruPageRE.FindAllStringIndex(s, -1)
	for i, st := range starts {
		end := len(s)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		var ws []figure.Word
		for _, m := range ruWordRE.FindAllStringSubmatch(s[st[0]:end], -1) {
			var w figure.Word
			w.X0, _ = strconv.ParseFloat(m[1], 64)
			w.Y0, _ = strconv.ParseFloat(m[2], 64)
			w.X1, _ = strconv.ParseFloat(m[3], 64)
			w.Y1, _ = strconv.ParseFloat(m[4], 64)
			w.Text = html.UnescapeString(m[5])
			ws = append(ws, w)
		}
		if len(ws) == 0 {
			continue
		}
		hs := make([]float64, len(ws))
		for k, w := range ws {
			hs[k] = w.Y1 - w.Y0
		}
		slices.Sort(hs)
		median := hs[len(hs)/2]
		cm := ruLatinCharmap(ws)
		k := cm == charmap.KOI8R
		koi8 = koi8 || k
		var line strings.Builder
		var prev figure.Word
		for n, w := range ws {
			t := w.Text
			if cm != nil {
				t = ruFromLatin(t, cm)
			}
			if w.Y1-w.Y0 < 0.8*median {
				t = ruSmall + t
			}
			switch {
			case n == 0:
			case w.X0 < prev.X0-10 || w.Y1-prev.Y1 > median || prev.Y1-w.Y1 > median:
				lines = append(lines, RuPDFLine{Text: line.String(), Page: i + 1})
				line.Reset()
			case w.X0-prev.X1 > 0.5:
				line.WriteString(" ")
			}
			line.WriteString(t)
			prev = w
		}
		lines = append(lines, RuPDFLine{Text: line.String(), Page: i + 1})
	}
	return lines, koi8
}

// ruLatinCharmap tells if the words of a page are the codes of a Russian 8-bit charset read as Latin-1: there are more letters from U+00C0 to U+00FF than Cyrillic letters. It gives the charset, KOI8-R or Windows-1251, that makes more small letters, as a Russian text has mostly small letters. It gives nil for a page of Unicode text.
func ruLatinCharmap(ws []figure.Word) *charmap.Charmap {
	latin, cyr := 0, 0
	var b strings.Builder
	for _, w := range ws {
		b.WriteString(w.Text)
		for _, r := range w.Text {
			switch {
			case r >= 0xc0 && r <= 0xff:
				latin++
			case unicode.Is(unicode.Cyrillic, r):
				cyr++
			}
		}
	}
	if latin <= cyr {
		return nil
	}
	lower := func(cm *charmap.Charmap) int {
		n := 0
		for _, r := range ruFromLatin(b.String(), cm) {
			if unicode.Is(unicode.Cyrillic, r) && unicode.IsLower(r) {
				n++
			}
		}
		return n
	}
	if lower(charmap.Windows1251) > lower(charmap.KOI8R) {
		return charmap.Windows1251
	}
	return charmap.KOI8R
}

// ruFromLatin puts the Latin-1 letters of a word back to the letters of the charset cm. The old TeX fonts give an en dash as { and an em dash as |, and a thin space as U+0082.
func ruFromLatin(s string, cm *charmap.Charmap) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0xa0 && r <= 0xff:
			b.WriteRune(cm.DecodeByte(byte(r)))
		case r == '{' && cm == charmap.KOI8R:
			b.WriteRune('–')
		case r == '|' && cm == charmap.KOI8R:
			b.WriteRune('—')
		case r == 0x82:
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	// ruPDFGrade is a heading line of a grade paper, as "8 класс 10.03.2013", "12 марта 2017 года • 8 класс" or "LXXIII Московская математическая олимпиада, 11 класс".
	ruPDFGrade = regexp.MustCompile(`(?:^|[^\p{L}\d])(\d{1,2})\s*класс(?:[^\p{L}]|$)`)
	// ruPDFMark is the start of a problem: "Задача 1." or "1.".
	ruPDFMark = regexp.MustCompile(`^\s*(?:Задача\s*)?(\d{1,2})\s*\.\s*`)
	// ruPDFEnd is a line after the last problem of a paper: a notice of the jury, an address or a heading of the olympiad.
	ruPDFEnd = regexp.MustCompile(`(?i)^[IVXLC]+\s|Подробн|Закрыти|состоится|https?:|www\.|^УСЛОВИЯ|^РЕШЕНИЯ|^Решени|^Ответ`)
	// ruFigureWord is a word of a text that points at a drawing.
	ruFigureWord = regexp.MustCompile(`(?i)рис\.|рисун|изображ`)
	// ruParticle is the start of a line that keeps the hyphen before it, as in "какой-\nто".
	ruParticle = regexp.MustCompile(`^(?:то|либо|нибудь|таки|ка)(?:[^\p{L}]|$)`)
	// ruItemLine is the start of an item, as "а)" or "б)".
	ruItemLine = regexp.MustCompile(`^[а-е]\)\s`)
	// ruPDFOdd is a sign that the text layer lost the form of a formula: a part of a big brace, a glyph with no letter, or a line of a number alone.
	ruPDFOdd = regexp.MustCompile(`[⎧⎨⎩⎪⎛⎝⎞⎠⎡⎣⎤⎦�]|(?m)^\s*[\d\s+=−-]{1,4}\s*$`)
	// ruKOI8Dot is a dot of a formula that the old TeX fonts give as a colon, as "de:" at the end of a sentence.
	ruKOI8Dot = regexp.MustCompile(`([A-Za-z0-9)]):(\s|$)`)
)

// RuPDF reads the problems of the text lines of a PDF of grade papers, as the papers of the Moscow olympiad. A paper starts at a heading line that names its grade and ends at the next one or at a notice of the jury. The first paper of a grade counts, as a PDF can print a paper twice. day2 tells that the PDF holds the second day of grade 11. koi8 tells that the text came from the old fonts, whose colon after a formula is a dot.
func RuPDF(lines []RuPDFLine, edition, url string, grades func(int) bool, day2, koi8 bool) []RuProblem {
	type paper struct {
		grade int
		lines []RuPDFLine
	}
	var papers []paper
	seen := map[int]bool{}
	cur := -1
	for _, l := range lines {
		plain := strings.ReplaceAll(l.Text, ruSmall, "")
		if m := ruPDFGrade.FindStringSubmatch(plain); m != nil && ruPDFMark.FindStringIndex(plain) == nil && utf8.RuneCountInString(plain) <= 90 {
			g, _ := strconv.Atoi(m[1])
			cur = -1
			if grades(g) && !seen[g] {
				seen[g] = true
				papers = append(papers, paper{grade: g})
				cur = len(papers) - 1
			}
			continue
		}
		if cur >= 0 {
			papers[cur].lines = append(papers[cur].lines, l)
		}
	}
	var out []RuProblem
	for _, pp := range papers {
		prefix := "p" + strconv.Itoa(pp.grade)
		if day2 && pp.grade == 11 {
			prefix = "d11"
		}
		var probs [][]string
		var pages []int
		ended := false
		for _, line := range pp.lines {
			l := line.Text
			plain := strings.ReplaceAll(l, ruSmall, "")
			if m := ruPDFMark.FindStringSubmatch(plain); m != nil {
				if n, _ := strconv.Atoi(m[1]); n == len(probs)+1 {
					probs = append(probs, []string{l})
					pages = append(pages, line.Page)
					ended = false
					continue
				}
			}
			if len(probs) == 0 || ended {
				continue
			}
			if ruPDFEnd.MatchString(plain) {
				ended = true
				continue
			}
			probs[len(probs)-1] = append(probs[len(probs)-1], l)
		}
		for i, pl := range probs {
			p := RuProblem{Edition: edition, Position: prefix + strconv.Itoa(i+1), URL: url, Page: pages[i]}
			raw := strings.Join(pl, "\n")
			p.Text = RuPDFText(pl, koi8)
			p.Figure = ruFigureWord.MatchString(raw)
			if strings.Contains(raw, ruSmall) || ruPDFOdd.MatchString(strings.ReplaceAll(strings.Join(pl[1:], "\n"), ruSmall, "")) {
				p.Flags = append(p.Flags, "layout")
			}
			out = append(out, p)
		}
	}
	return out
}

// RuPDFText makes the lines of one problem of a PDF into its text: the mark of the problem and the author go, a word cut by a hyphen at the end of a line is joined, an item starts a new line, and the math goes into TeX.
func RuPDFText(lines []string, koi8 bool) string {
	var b strings.Builder
	for i, l := range lines {
		l = strings.TrimSpace(strings.ReplaceAll(l, ruSmall, ""))
		if i == 0 {
			l = ruPDFMark.ReplaceAllString(l, "")
		}
		if l == "" {
			continue
		}
		if b.Len() > 0 {
			s := b.String()
			switch {
			case ruItemLine.MatchString(l):
				b.WriteString("\n")
			case strings.HasSuffix(s, "-") && !strings.HasSuffix(s, " -") && ruLowerStart(l) && !ruParticle.MatchString(l):
				b.Reset()
				b.WriteString(strings.TrimSuffix(s, "-"))
			case strings.HasSuffix(s, "-") && !strings.HasSuffix(s, " -") && ruLowerStart(l):
			default:
				b.WriteString(" ")
			}
		}
		b.WriteString(l)
	}
	s := b.String()
	if koi8 {
		s = ruKOI8Dot.ReplaceAllString(s, "$1.$2")
	}
	s = strings.NewReplacer("– –", "—", "{", "\x01", "}", "\x02").Replace(s)
	s = RuPlain(s)
	s = ruNote.ReplaceAllString(s, "")
	s = ruAuthor.ReplaceAllString(s, "")
	return strings.NewReplacer("\x01", "{", "\x02", "}").Replace(RuTeX(s))
}

// ruLowerStart tells if a line starts with a small Cyrillic letter.
func ruLowerStart(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.Is(unicode.Cyrillic, r) && unicode.IsLower(r)
}
