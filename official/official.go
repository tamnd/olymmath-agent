// Package official reads the official papers of a competition and anchors the problems of the content to them (spec 05.7, the native lane). A PDF text layer loses the layout of formulas, so the text of a paper is not a canonical statement. It is evidence: when the words of the paper agree with the canonical text of a problem, the problem gets a statement of class H1 that links to the paper and its page. When they do not agree, the problem goes to the review queue.
package official

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
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
}

// Segment is the text of one problem in a paper.
type Segment struct {
	Position string
	// Page is the page of the paper where the problem starts, from 1.
	Page int
	Text string
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
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	text := out.String()
	var ascii, all int
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsSymbol(r) {
			all++
			if r < unicode.MaxASCII {
				ascii++
			}
		}
	}
	if ascii*2 < all {
		return "", ErrNoText
	}
	return text, nil
}

// markerRE finds the start of a problem: "Problem 3." or "1959/3." at the start of a line.
var markerRE = regexp.MustCompile(`(?m)^\s*(?:Problem\s+([0-9]{1,2})\s*[.:]|(?:19|20)[0-9]{2}\s*/\s*([0-9]{1,2})\s*\.)`)

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
		out = append(out, Segment{Position: pos, Page: strings.Count(text[:m[0]], "\f") + 1, Text: clean(text[m[1]:end])})
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

// Papers holds the segments of the papers of each edition.
type Papers map[string][]Segment

// Locate gives the position of a text in the paper of an edition: the position of the segment with the best word signal. It gives "" when the edition has no paper, when the best signal is less than Least, or when the best signal of another position is less than Margin below the best. Two segments of one position, for example two copies of one text, do not compete.
func (ps Papers) Locate(edition, text string) string {
	best := map[string]float64{}
	for _, sg := range ps[edition] {
		if t := Signal(text, sg.Text); t > best[sg.Position] {
			best[sg.Position] = t
		}
	}
	top, second, pos := 0.0, 0.0, ""
	for p, t := range best {
		switch {
		case t > top || (t == top && p < pos):
			top, second, pos = t, top, p
		case t > second:
			second = t
		}
	}
	if top < Least || top-second < Margin {
		return ""
	}
	return pos
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
func numbers(s string) []string {
	out := numberRE.FindAllString(s, -1)
	slices.Sort(out)
	return slices.Compact(out)
}

// Trace tells if text is a transcription of the text of a paper: the order signal is at least TraceLeast and the two texts have the same set of numbers. The old papers set the point O so that the text layer gives 0, so a 0 that only the paper has is not counted.
func Trace(paper, text string) bool {
	if Order(paper, text) < TraceLeast {
		return false
	}
	np, nt := numbers(paper), numbers(text)
	if !slices.Contains(nt, "0") {
		np = slices.DeleteFunc(np, func(n string) bool { return n == "0" })
	}
	return slices.Equal(np, nt)
}
