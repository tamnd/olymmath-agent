// Package official reads the official papers of a competition and anchors the problems of the content to them (spec 05.7, the native lane). A PDF text layer loses the layout of formulas, so the text of a paper is not a canonical statement. It is evidence: when the words of the paper agree with the canonical text of a problem, the problem gets a statement of class H1 that links to the paper and its page. When they do not agree, the problem goes to the review queue.
package official

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Version names the rules of this package.
const Version = "official/1"

// Agree is the lowest word signal at which a paper text and a canonical text are one problem. A lower signal gives a review item.
const Agree = 0.5

// Paper is one official paper of one edition.
type Paper struct {
	// Edition is the locator of the edition, for example imo/2024.
	Edition string
	URL     string
	Lang    string
	Name    string
	// Booklet is true for a shortlist booklet, whose text SplitBooklet cuts.
	Booklet bool
}

// Segments cuts the text of the paper into problems.
func (p Paper) Segments(text string) []Segment {
	if p.Booklet {
		return SplitBooklet(text)
	}
	return Split(text)
}

// Segment is the text of one problem in a paper.
type Segment struct {
	// Position is the position of the problem in the edition. It is "" for a problem of a paper whose problems have no position in the content, such as the paper of an earlier stage of a national olympiad. Such a segment only competes with the others in Locate.
	Position string
	// Page is the page of the paper where the problem starts, from 1.
	Page int
	Text string
	// Lang is the code of the language of the paper. It is "" for English.
	Lang string
	// URL is the address of the paper. LoadEditions sets it.
	URL string
}

// Compare gives the signal of a text in English and a segment: the word signal when the paper is in English, or else the formula signal.
func Compare(text string, sg Segment) float64 {
	if English(sg.Lang) {
		return Signal(text, sg.Text)
	}
	return FormulaSignal(text, sg.Text, sg.Lang)
}

// IMO gives the English papers of the IMO from one year to another. There was no IMO in 1980.
func IMO(from, to int) []Paper {
	var out []Paper
	for y := from; y <= to; y++ {
		if y == 1980 {
			continue
		}
		out = append(out, Paper{
			Edition: "imo/" + strconv.Itoa(y),
			URL:     fmt.Sprintf("https://www.imo-official.org/assets/documents/problems/%d/%d_eng.pdf", y, y),
			Lang:    "en",
			Name:    fmt.Sprintf("IMO %d official English paper", y),
		})
	}
	return out
}

// ErrNoText is the error when the text layer of a PDF is not readable, for example when a font has no map to Unicode. Such a paper needs the vision lane.
var ErrNoText = errors.New("the text layer of the PDF is not readable")

// Text runs pdftotext on the bytes of a PDF and returns the text, with a form feed between pages. It returns ErrNoText when less than half of the letters are ASCII letters.
func Text(ctx context.Context, pdf []byte) (string, error) {
	text, err := pdftotext(ctx, pdf, "-layout")
	if err != nil {
		return "", err
	}
	if !readable(text) {
		return "", ErrNoText
	}
	return text, nil
}

// markerRE finds the start of a problem: "Problem 3." or "1959/3." at the start of a line. The German word Aufgabe and the Polish word Zadanie are markers too, and a marker with a word can be alone on its line, as in "Aufgabe 3".
var markerRE = regexp.MustCompile(`(?m)^\s*(?:(?:Problem|Aufgabe|Zadanie)\s+([0-9]{1,2})(?:\s*[.:]|[ \t]*$)|(?:19|20)[0-9]{2}\s*/\s*([0-9]{1,2})\s*\.)`)

// bareRE finds the start of a problem in a paper with no word before the number: "3." at the start of a line.
var bareRE = regexp.MustCompile(`(?m)^\s*()([0-9]{1,2})\.\s`)

// Split cuts the text of a paper into problems. A problem that has a marker twice, for example in two languages, keeps its first text. When the paper has no marker with a word or a year, Split uses the numbers at the start of a line, but only in the sequence 1, 2, 3 and so on, because a list in a problem also has numbers.
func Split(text string) []Segment {
	ms := markerRE.FindAllStringSubmatchIndex(text, -1)
	if len(ms) == 0 {
		next := 1
		for _, m := range bareRE.FindAllStringSubmatchIndex(text, -1) {
			if n, _ := strconv.Atoi(text[m[4]:m[5]]); n == next {
				ms = append(ms, m)
				next++
			}
		}
	}
	var out []Segment
	seen := map[string]bool{}
	for i, m := range ms {
		var n string
		if m[3] > m[2] {
			n = text[m[2]:m[3]]
		} else {
			n = text[m[4]:m[5]]
		}
		end := len(text)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		pos := "p" + strings.TrimLeft(n, "0")
		if seen[pos] {
			continue
		}
		seen[pos] = true
		// The marker can start with the form feed of its page, so the page counts the form feeds up to the end of the marker.
		out = append(out, Segment{Position: pos, Page: strings.Count(text[:m[1]], "\f") + 1, Text: clean(text[m[1]:end])})
	}
	return out
}

// footerRE matches the lines that a paper repeats on each page, such as "Language: English" and "Day: 1".
var footerRE = regexp.MustCompile(`(?im)^\s*(language:.*|day:?\s*[12]\s*|time:.*|each problem is worth.*|.*\b(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b,.*[0-9]{4}\s*)$`)

func clean(s string) string {
	s = strings.ReplaceAll(s, "\f", "\n")
	s = footerRE.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// Words gives the words of a text for the word signal: each run of three or more letters, in lower case. A TeX command, such as \frac, is not a word.
func Words(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range wordList(s) {
		out[w] = true
	}
	return out
}

// wordList gives the words of a text in their order.
func wordList(s string) []string {
	var out []string
	rs := []rune(strings.ToLower(s))
	for i := 0; i < len(rs); {
		if rs[i] == '\\' {
			i++
			for i < len(rs) && unicode.IsLetter(rs[i]) {
				i++
			}
			continue
		}
		if !unicode.IsLetter(rs[i]) {
			i++
			continue
		}
		j := i
		for j < len(rs) && unicode.IsLetter(rs[j]) {
			j++
		}
		if j-i >= 3 {
			out = append(out, string(rs[i:j]))
		}
		i = j
	}
	return out
}

// Signal gives the word signal of two texts: the Jaccard index of their words.
func Signal(a, b string) float64 {
	wa, wb := Words(a), Words(b)
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	n := 0
	for w := range wa {
		if wb[w] {
			n++
		}
	}
	return float64(n) / float64(len(wa)+len(wb)-n)
}

// Least is the lowest word signal for which Locate gives a position. It is lower than Agree, because Locate compares a text with the few problems of one paper, and the margin to the second best signal must also be clear.
const Least = 0.3

// Margin is the smallest difference between the best and the second best word signal for which Locate gives a position.
const Margin = 0.15

// FormulaMargin is the smallest difference between the best and the second best formula signal for which Locate gives a position. It is larger than Margin, because two problems share more formula tokens, such as 1, 2 and n, than words. On the papers of pl-om and de-bundeswettbewerb, with the papers of the earlier stages of pl-om, the rows of hf-ai-mo-aops that Locate placed agreed with each of the 60 rows whose position was known, and no row of an earlier stage got a position.
const FormulaMargin = 0.2

// Papers holds the segments of the papers of each edition.
type Papers map[string][]Segment

// Locate gives the position of a text in the papers of an edition: the position of the segment with the best signal (see Compare). It gives "" when the edition has no paper, when the best signal is less than Least, when the best signal of another position is less than Margin below the best, or when the best segment has no position. The margin is FormulaMargin when the best segment is not in English. Two segments of one position, for example two copies of one text, do not compete.
func (ps Papers) Locate(edition, text string) string {
	best := map[string]float64{}
	english := map[string]bool{}
	for _, sg := range ps[edition] {
		if t := Compare(text, sg); t > best[sg.Position] || (t == best[sg.Position] && English(sg.Lang)) {
			best[sg.Position], english[sg.Position] = t, English(sg.Lang)
		}
	}
	top, second, pos := 0.0, 0.0, ""
	found := false
	for p, t := range best {
		switch {
		case !found || t > top || (t == top && p < pos):
			if found {
				second = max(second, top)
			}
			top, pos, found = t, p, true
		case t > second:
			second = t
		}
	}
	margin := Margin
	if !english[pos] {
		margin = FormulaMargin
	}
	if top < Least || top-second < margin {
		return ""
	}
	return pos
}

// Disagrees tells if the English paper of an edition has another problem at the position pos than the text: the paper has a segment at pos in English, the word signal of the text and that segment is less than Least, and Locate does not place the text at pos. A paper in another language never disagrees, because the formula signal is too weak to refuse a position. An edition with no segment at pos never disagrees.
func (ps Papers) Disagrees(edition, text, pos string) bool {
	found := false
	for _, sg := range ps[edition] {
		if sg.Position != pos || !English(sg.Lang) {
			continue
		}
		if Signal(text, sg.Text) >= Least {
			return false
		}
		found = true
	}
	return found && ps.Locate(edition, text) != pos
}

// TraceLeast is the lowest order signal at which a transcription is traced to the text of a paper.
const TraceLeast = 0.9

// Order gives the order signal of two texts: twice the length of the longest common sequence of their words, over the number of words of both. Unlike the word signal, it counts each word as often as it occurs and in its place.
func Order(a, b string) float64 {
	wa, wb := wordList(a), wordList(b)
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	d := make([]int, len(wb)+1)
	for i := range wa {
		prev := 0
		for j := range wb {
			up := d[j+1]
			switch {
			case wa[i] == wb[j]:
				d[j+1] = prev + 1
			case d[j] > d[j+1]:
				d[j+1] = d[j]
			}
			prev = up
		}
	}
	return 2 * float64(d[len(wb)]) / float64(len(wa)+len(wb))
}

var numberRE = regexp.MustCompile(`[0-9]+`)

// numbers gives the set of numbers of a text, sorted.
// urlRE matches a URL, for example the address of a figure. Its digits are not numbers of the text.
var urlRE = regexp.MustCompile(`https?://[^\s)\]]+`)

// notEqual is the text that the text layer of many old papers gives for the sign ≠.
var notEqual = strings.NewReplacer("6=", "≠")

func numbers(s string) []string {
	out := numberRE.FindAllString(urlRE.ReplaceAllString(s, " "), -1)
	slices.Sort(out)
	return slices.Compact(out)
}

// SameNumbers tells if a text and the text of a paper have the same set of numbers. A 0 that only the paper has is not counted, see Trace, and the text 6= of the paper is the sign ≠.
func SameNumbers(paper, text string) bool {
	np, nt := numbers(notEqual.Replace(paper)), numbers(text)
	if !slices.Contains(nt, "0") {
		np = slices.DeleteFunc(np, func(n string) bool { return n == "0" })
	}
	return slices.Equal(np, nt)
}

// NumberDiff gives the numbers that only one of the two texts has. A number of the paper has the prefix "paper:", and a number of the text has the prefix "text:".
func NumberDiff(paper, text string) []string {
	np, nt := numbers(notEqual.Replace(paper)), numbers(text)
	var out []string
	for _, n := range np {
		if !slices.Contains(nt, n) && (n != "0" || slices.Contains(nt, "0")) {
			out = append(out, "paper:"+n)
		}
	}
	for _, n := range nt {
		if !slices.Contains(np, n) {
			out = append(out, "text:"+n)
		}
	}
	return out
}

// Trace tells if text is a transcription of the text of a paper: the order signal is at least TraceLeast and the two texts have the same set of numbers. The old papers set the point O so that the text layer gives 0, so a 0 that only the paper has is not counted.
func Trace(paper, text string) bool {
	return Order(paper, text) >= TraceLeast && SameNumbers(paper, text)
}
