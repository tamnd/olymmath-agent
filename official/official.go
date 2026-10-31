// Package official reads the official papers of a competition and anchors the problems of the content to them (spec 05.7, the native lane). A PDF text layer loses the layout of formulas, so the text of a paper is not a canonical statement. It is evidence: when the words of the paper agree with the canonical text of a problem, the problem gets a statement of class H1 that links to the paper and its page. When they do not agree, the problem goes to the review queue.
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

// Text runs pdftotext on the bytes of a PDF and returns the text, with a form feed between pages.
func Text(ctx context.Context, pdf []byte) (string, error) {
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// markerRE finds the start of a problem: "Problem 3." or "1959/3." at the start of a line.
var markerRE = regexp.MustCompile(`(?m)^\s*(?:Problem\s+([0-9]{1,2})\s*[.:]|(?:19|20)[0-9]{2}\s*/\s*([0-9]{1,2})\s*\.)`)

// Split cuts the text of a paper into problems. A problem that has a marker twice, for example in two languages, keeps its first text.
func Split(text string) []Segment {
	ms := markerRE.FindAllStringSubmatchIndex(text, -1)
	var out []Segment
	seen := map[string]bool{}
	for i, m := range ms {
		var n string
		if m[2] >= 0 {
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
			out[string(rs[i:j])] = true
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
