// Package ocr reads the text of a scanned official paper with a vision model. It renders each page to an image, asks a model for the text of the page as Markdown with TeX math, checks and repairs the answer, and cuts the text into problems. Each answer is kept in a cache on disk under the hash of the page image, so a second run of the same pages calls no model.
//
// The model only reads. The words of the text are the words of the paper, so the text keeps the class of the paper, but each text records that a model read it (spec 05.7), and no text that a model read is shown as a text that a person checked.
package ocr

import (
	"regexp"
	"strings"

	"github.com/tamnd/olymmath-agent/normalize"
)

// PromptVersion names the prompt and the rules of this package. A change to the prompt changes it, so the cache does not give an answer to an old prompt.
const PromptVersion = "ocr/1"

// Empty is the answer of the model for a page with no text.
const Empty = "EMPTY PAGE"

// Prompt is the instruction for one page.
const Prompt = `The image is one page of an official paper of a mathematics olympiad. Write the text of the page as Markdown with TeX math.

Rules:
- Write the words exactly as they are printed, in the language of the page. Do not translate. Do not correct. Do not shorten. Do not add words.
- Write all mathematics in TeX: $...$ in a line, and $$...$$ for a formula on its own line. Use only commands that KaTeX can render.
- Write each heading on its own line that starts with "# ". A heading is a title, a round, a day, a date, a time, or instructions for the contestants.
- Start each problem on a new line with its number as printed, in bold, as **1.** When the number has a word before it, as in "Problem 1" or "Zadanie 1", write the word too, as **Problem 1.**
- Write the word Solution, Answer or Hint, in the language of the page, on its own line that starts with "# ", before each solution, answer or hint.
- For a figure or a diagram, write [figure] on its own line.
- When you cannot read a word or a symbol, write [?] in its place.
- When the image shows two pages side by side, write the left page first and then the right page.
- Do not write page numbers, running heads or footers.
- When the page has no text, write only: EMPTY PAGE
- Write only the Markdown. Do not write comments or code fences.`

// CropPromptVersion names CropPrompt, as PromptVersion names Prompt. The cache key of a cropped image holds it, so a crop and a page never share an answer.
const CropPromptVersion = "ocr-crop/1"

// CropPrompt is the instruction for an image that is a part of a page and holds one problem, as the images of a dataset that cut a paper into problems. Such an image often shows, at its top or bottom edge, a line of the problem before or after it that is cut through. The model leaves that line out, so it does not mark it with [?].
const CropPrompt = `The image is a part of one page of an official paper of a mathematics olympiad. It holds one problem. Write the text of the problem as Markdown with TeX math.

Rules:
- Write the words exactly as they are printed, in the language of the page. Do not translate. Do not correct. Do not shorten. Do not add words.
- Write all mathematics in TeX: $...$ in a line, and $$...$$ for a formula on its own line. Use only commands that KaTeX can render.
- Start the problem on a new line with its number as printed, in bold, as **1.** When the number has a word before it, as in "Problem 1" or "Soru 1", write the word too, as **Soru 1.**
- Write the choices of a multiple-choice problem as they are printed, after the problem.
- A line at the top or the bottom edge of the image that is cut through, so that only a part of its letters shows, belongs to another problem or to an answer. Do not write it, and do not write [?] for it.
- For a figure or a diagram, write [figure] on its own line.
- When you cannot read a word or a symbol in a whole line, write [?] in its place.
- Do not write page numbers, running heads or footers.
- When the image has no text, write only: EMPTY PAGE
- Write only the Markdown. Do not write comments or code fences.`

// Flags of the checks. A flag lowers the confidence of the text that has it.
const (
	FlagFence      = "fence"      // the answer was in a code fence
	FlagRefusal    = "refusal"    // the answer is a refusal or a comment, not the page
	FlagDollars    = "dollars"    // a $ has no pair
	FlagBraces     = "braces"     // a { has no pair
	FlagUnreadable = "unreadable" // the model marked a word with [?]
	FlagFigure     = "figure"     // the text has a figure that the text does not hold
	FlagShort      = "short"      // the text is too short to be a problem
)

var fenceRE = regexp.MustCompile("(?s)^\\s*```[a-zA-Z]*\\s*\\n(.*?)\\n?```\\s*$")

// refusals are phrases that start an answer that is not the text of a page.
var refusals = []string{"i can't", "i cannot", "i'm unable", "i am unable", "i'm sorry", "i am sorry", "as an ai", "unfortunately, i", "here is the", "here's the", "the image shows", "the image contains"}

// Clean takes a code fence off an answer and trims it. The flags tell what it found.
func Clean(answer string) (string, []string) {
	var flags []string
	s := strings.TrimSpace(strings.ReplaceAll(answer, "\r\n", "\n"))
	if m := fenceRE.FindStringSubmatch(s); m != nil {
		s, flags = strings.TrimSpace(m[1]), append(flags, FlagFence)
	}
	low := strings.ToLower(s)
	for _, r := range refusals {
		if strings.HasPrefix(low, r) {
			flags = append(flags, FlagRefusal)
			break
		}
	}
	return s, flags
}

// Check gives the flags of a text: the math delimiters and the braces that have no pair, the words the model could not read, and the figures.
func Check(s string) []string {
	var flags []string
	if odd(s) {
		flags = append(flags, FlagDollars)
	}
	if !braces(s) {
		flags = append(flags, FlagBraces)
	}
	if strings.Contains(s, "[?]") {
		flags = append(flags, FlagUnreadable)
	}
	if strings.Contains(strings.ToLower(s), "[figure]") {
		flags = append(flags, FlagFigure)
	}
	return flags
}

// odd tells if the text has a $ with no pair. A $$ counts as one delimiter of its own kind, and \$ is a dollar sign and not a delimiter.
func odd(s string) bool {
	s = strings.ReplaceAll(s, `\$`, "")
	double := strings.Count(s, "$$")
	single := strings.Count(strings.ReplaceAll(s, "$$", ""), "$")
	return double%2 != 0 || single%2 != 0
}

// braces tells if each { has its }. A \{ and a \} are letters and do not count.
func braces(s string) bool {
	s = strings.NewReplacer(`\{`, "", `\}`, "").Replace(s)
	depth := 0
	for _, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// delimiters changes the LaTeX delimiters \( \) and \[ \] to the dollar delimiters that the site renders.
var delimiters = strings.NewReplacer(`\(`, "$", `\)`, "$", `\[`, "$$", `\]`, "$$")

var blankRun = regexp.MustCompile(`\n{3,}`)

// Repair makes the fixes that need no judgement: the dollar delimiters, the spaces at the ends of lines, and the runs of blank lines. The fixed text must keep each number, variable and operator, by the no-rewrite guard of spec 06.4 step 6, or Repair gives the text as it was.
func Repair(s string) string {
	fixed := delimiters.Replace(s)
	lines := strings.Split(fixed, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	fixed = strings.TrimSpace(blankRun.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
	// The share of the guard is for the fixes of a model. These fixes change only delimiters and spaces, so on a short text the share can be high while no number, variable or operator changes. The flags are the test that holds here.
	if c := normalize.Guard(s, fixed); len(c.Flags) > 0 {
		return strings.TrimSpace(s)
	}
	return fixed
}

// Base is the confidence of a text that passes every check. It is not 1, because no person compared the text with the page.
const Base = 0.9

// Threshold is the lowest confidence of a text that the content holds. A text below it gives a problem that only links to its page.
const Threshold = 0.8

// Confidence gives the confidence of a text with the flags. It is a score of the checks of this package, and not a measured probability. Each flag except a fence takes the text below Threshold.
func Confidence(flags []string) float64 {
	c := Base
	for _, f := range flags {
		switch f {
		case FlagFence:
			c -= 0.02
		case FlagRefusal:
			c = 0
		default:
			c -= 0.3
		}
	}
	return max(c, 0)
}
