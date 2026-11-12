// Package loose imports the seed rows that name no position as loose problems (spec 06.8). It cleans the text of each row, drops a row that is not one clear problem, and matches each row with the problems of the content and with the rows before it.
package loose

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/normalize"
)

// MaxText is the longest text, in characters, that the import takes. A longer row is in most cases a whole paper, or a problem with its solution.
const MaxText = 4000

// Reasons why a row is dropped.
const (
	TooShort  = "too short"
	TooLong   = "too long"
	NotStated = "not a statement"
	Solution  = "solution only"
	Contents  = "table of contents"
	Stub      = "stub"
	Request   = "request"
	Context   = "refers to another problem"
	Figure    = "needs a figure"
	Glued     = "several problems"
	BrokenTeX = "broken TeX"
	NoLang    = "unknown language"
	WebLink   = "links to a page"
	Template  = "a template"
	Lost      = "formulas lost"
	Science   = "physics or chemistry"
	Shortlist = "a shortlist or longlist row"
)

var (
	// headingRE is a heading line at the start of a text, such as "# Task 3." or "## Problem 5", with no formula.
	headingRE = regexp.MustCompile(`^\s*#{1,6}\s*[^\n$]{0,40}\n`)
	// leadRE is a problem mark at the start of a text, such as "Problem 4.", "**Problem 3.**", "Task 2:", "P6.", "7.", "11.4." or "2)".
	leadRE = regexp.MustCompile(`(?i)^\s*(?:\*\*|__)?\s*(?:(?:problem|task|question|exercise|задача)\s*-?\s*|[pq]\s*\.?\s*)?([1-9][0-9]?)((?:\.[0-9]{1,2})*)\s*(?:\.\)|[.:)])\s*(?:\*\*|__)?(?:\s|$)`)
	// tightLeadRE is a problem number glued to the text, as "3-There is" or "1.Let".
	tightLeadRE = regexp.MustCompile(`^\s*([1-9][0-9]?)[-.]\p{Lu}\p{Ll}`)
	// markRE is a problem mark inside a text: "Problem 2", "Task 3" or "p2." at the start of a line, or a bold "**3.**" or "**p14.**" anywhere. A row with such a mark after its start holds more than one problem. A post of a whole test often runs its problems on one line, as "... divided by $2024$ .**p14.** A right square pyramid ...".
	markRE = regexp.MustCompile(`(?im)(?:^|\n)\s*(?:\*\*|__)?\s*(?:(?:problem|task|задача)\s*[0-9]{1,2}\b|p[0-9]{1,2}\s*[.:)])|\*\*\s*p?[0-9]{1,2}\s*[.:)]\s*\*\*`)
	// numLineRE is a line that starts with a number and a point, such as "4. Find".
	numLineRE = regexp.MustCompile(`(?m)^\s*([1-9][0-9]?)\.\s`)
	// answerRE is the start of an answer or a solution that a dataset joined to the statement, on a line of its own.
	answerRE = regexp.MustCompile(`(?im)^\s*(?:\*\*|__)?\s*(?:answer|answers|otbet|ответ|solution|решение|proof|hint|указание|comment)\s*(?:\*\*|__)?\s*[.:]`)
	// solutionRE is a text that starts as a solution, not as a problem.
	solutionRE = regexp.MustCompile(`(?i)^\s*(?:#{1,6}\s*)?(?:[0-9]{1,2}\.\s*)?(?:\*\*|__)?\s*(?:(?:(?:a|an|the)\s+)?(?:my|another|quick|short|simple|alternative|alternate|nice|easy|official|elegant|full|complete)\s+)?(?:solution|sol\.|proof|answer|решение|ответ|otbet)\b`)
	// stepRE is a text that starts with a step of a solution, as "Since the right side is divisible by 49, ..." or "By the condition, ...". Some datasets give the solution of a problem as its statement.
	stepRE = regexp.MustCompile(`^\s*(?:By the condition|From the condition|Since\b|Hence\b|Therefore\b|Thus\b|From this\b|Then\b)`)
	// templateRE is a parameter of a template, as in "Given a regular param1." A template stands for several problems, which a table at its end gives.
	templateRE = regexp.MustCompile(`(?i)\bparam\s?[0-9]+\b`)
	// lostRE is a sentence that a blank line cuts where a dataset dropped a displayed formula, as in "Prove that if the equation\n\nhas three real roots, then", or a note of a translator that says so.
	lostRE = regexp.MustCompile(`(?i)[a-z,]\n\n+(?:where|then|holds?|is|are|has|have|and|for|divides|takes|to|gdzie|dla|oraz|jest|ma)\b|\b(?:not (?:provided|given|included) in the (?:original )?text|original text had|untranslated)\b`)
	// scienceRE is a term of physics or chemistry. Problems of physics and of the other sciences are out of scope (spec 02); some olympiad sets mix them with the problems of mathematics. A puzzle that only names physics, as a club of physics, does not match.
	scienceRE = regexp.MustCompile(`(?i)\((?:physics|chemistry)\)|\b(?:refractive|refraction|coefficient of friction|friction(?:al)? force|voltage|resistors?|resistivity|internal resistance|capacitors?|capacitance|piston|heat capacity|specific heat|latent heat|buoyan\w*|kinetic energy|potential energy|pendulum|wavelength|joules?|ohms?|electronegativity|hybridi[sz]ed|hybridization|moles? of|molar mass|(?:sulfuric|sulphuric|nitric|hydrochloric|hydroiodic|orthophosphoric|phosphoric|acetic) acid|isotopes?|acceleration due to gravity|air resistance|thermostat|spring constant|stiffness(?:es)? of the springs?|amperes?|ammeter|voltmeter|galvanic|emf|electromotive|ideal gas|(?:converging|diverging|collecting|thin) lens|focal length|optical (?:power|axis|center|centre)|electric (?:field|current|charge|circuit)|electrical circuit|current source|precipitat\w*|aqueous|alkanes?|alkynes?|alkenes?|gravitational forces?|elastic collision)\b`)
	// shortlistRE is the mark of a problem of the IMO shortlist or longlist in a compendium: the country that proposed it, as "(GBR 2)" or "A2 (GBR)" at the start, or a note that the problem was set at the IMO, as "${ }^{\mathrm{IMO} 5}$". The row does not give the year, and the content has the shortlists, the longlists and the IMO by position, so such a row is left to them.
	shortlistRE = regexp.MustCompile(`^\s*(?:[IVX]{1,3}\s+[0-9]{1,2}\s+)?(?:[ACGNS][0-9]{1,2}\s+)?\([A-Z]{3}(?:\s+[0-9]{1,2})?\)|\^\{\s*\\(?:mathrm|text)\s*\{\s*IMO`)
	// personRE is a line of a poster about himself. With no formula, no question and no task it marks a forum post that is news and not a problem.
	personRE = regexp.MustCompile(`\bI(?:'m| am| have|'ve| will|'ll| plan)\b|\bsorry\b`)
	// taskRE is a word that asks for something or states a claim. A text with none of them, once a cut leaves it or with no formula, is not a problem: "Given an odd number $m$." is what is left of "Given an odd number $m$. Proof: for any $c$ ...", where Proof stands for Prove.
	taskRE = regexp.MustCompile(`(?i)\?|\b(?:prove|proof|show|find|determine|compute|calculate|evaluate|solve|what|how|which|whether|is there|are there|does|do|can|could|must|exists?|construct|describe|decide|give|simplify|express|count|demonstrate|verify|then|if|when|holds?|restore|reconstruct|mark|cut|inscribe|transform|draw|split|divide|arrange|fill|colou?r|obtain|define|consider|propose|discuss|build|design|let)\b|_{3,}|\b(?:докаж|найд|определ|сколько|можно|верно)`)
	// contentsRE finds the lines of a table of contents: a dotted leader or the word contents.
	contentsRE = regexp.MustCompile(`(?i)\.{5,}|\\ldots\s*\\ldots\s*\\ldots|\btable of contents\b|^\s*contents\s*$`)
	// stubRE names a fact of the text around the row, such as "Prove the lemma." or "Prove Claim 2.". A row with such a name, no formula and few words is a stub.
	stubRE = regexp.MustCompile(`(?i)\b(?:lemma|claim|corollary|proposition)\b`)
	// requestRE is a post that asks for help or for a translation.
	requestRE = regexp.MustCompile(`(?i)\b(?:please|pls|plz)\b[^.?!]{0,40}\b(?:help|translate|solve|post|check)\b|\bhelp me\b|\bappreciate (?:any |some )?help\b|\bcan (?:someone|anyone|somebody)\b|\b(?:anyone|somebody|someone) (?:can|could|help|knows?)\b|\bany (?:hints?|ideas?|help)\b|\bthank(?:s| you)\b|\btranslat(?:e|ion) (?:this|please|needed|request)|\bi (?:need|want) (?:a |the )?(?:solution|help|hint)|\bhow (?:do|can|should) i\b|\bi (?:can't|cannot|couldn't|could not) (?:solve|prove|find)\b|\bmy (?:teacher|friend|solution|attempt)\b`)
	// contextRE is a text that depends on another problem.
	contextRE = regexp.MustCompile(`(?i)\b(?:previous|preceding|above|last|prior) (?:problem|exercise|question|task)\b|\b(?:problem|exercise|task) [0-9]{1,3} (?:above|before)\b|\bas in (?:problem|exercise|task) [0-9]{1,3}\b|\bfrom (?:problem|exercise|task) [0-9]{1,3}\b|\bof (?:the )?(?:previous|preceding) part\b`)
	// figureRE is a text that needs a figure that the row does not have.
	figureRE = regexp.MustCompile(`(?i)\[asy\]|\bfigure shown\b|\bas shown\b|\bthe (?:diagram|picture|photo)\b|\bphoto\b|\(see (?:the )?fig|\bfig\.\s*[0-9]|\[img\]|!\[|<img|\\includegraphics|https?://\S+\.(?:png|jpe?g|gif|svg)\b|\b(?:see|in|on) (?:the )?(?:figure|picture|diagram|drawing|image)\b|\b(?:figure|picture|diagram|drawing) (?:below|above|shows|on the right|on the left)\b|\bshown (?:below|above|in the (?:figure|picture|diagram))\b|\bpictured\b|\b(?:рисун\p{L}*|рис\.)`)
)

// Clean gives the text of a row as a statement, or "" and the reason why the row is not one clear problem. The text is normalized (spec 06.4), a heading line and a problem mark at the start are removed, and an answer or a solution that the dataset joined to the end is cut.
func Clean(raw string) (string, string) {
	plain := markup(raw)
	text := normalize.Statement(plain, "")
	// cut tells if an answer or a solution was cut from the end of the text.
	cut := len(normalize.CutTail(plain)) < len(strings.TrimSpace(plain))
	if solutionRE.MatchString(text) {
		return "", Solution
	}
	if m := headingRE.FindStringIndex(text); m != nil && !strings.ContainsAny(text[:m[1]], "?") {
		text = strings.TrimSpace(text[m[1]:])
	}
	lead := 0
	if m := leadRE.FindStringSubmatchIndex(text); m != nil {
		lead, _ = strconv.Atoi(text[m[2]:m[3]])
		text = strings.TrimSpace(text[m[1]:])
	} else if m := tightLeadRE.FindStringSubmatchIndex(text); m != nil {
		lead, _ = strconv.Atoi(text[m[2]:m[3]])
		text = strings.TrimSpace(text[m[3]+1:])
	}
	// A step of a solution and the mark of the compendium often follow a problem number, as in "5. Since ..." or "27. (BLR) Ten points ...", so they are read after the number is cut.
	switch {
	case stepRE.MatchString(text):
		return "", Solution
	case shortlistRE.MatchString(text):
		return "", Shortlist
	}
	if m := answerRE.FindStringIndex(text); m != nil {
		if m[0] < canon.MinText {
			return "", Solution
		}
		text = strings.TrimSpace(text[:m[0]])
		cut = true
	}
	if markRE.MatchString(text) {
		return "", Glued
	}
	if lead > 0 {
		// A row that starts with problem 3 and has a line that starts with 4. holds the next problem too.
		for _, m := range numLineRE.FindAllStringSubmatch(text, -1) {
			if n, _ := strconv.Atoi(m[1]); n == lead+1 {
				return "", Glued
			}
		}
	}
	n := utf8.RuneCountInString(text)
	switch {
	case n < canon.MinText:
		return "", TooShort
	case n > MaxText:
		return "", TooLong
	case contentsRE.MatchString(text):
		return "", Contents
	case stubRE.MatchString(text) && !strings.Contains(text, "$") && n < 150:
		return "", Stub
	case requestRE.MatchString(text):
		return "", Request
	case contextRE.MatchString(text):
		return "", Context
	case templateRE.MatchString(text):
		return "", Template
	case lostRE.MatchString(text):
		return "", Lost
	case scienceRE.MatchString(text):
		return "", Science
	case figureRE.MatchString(text):
		return "", Figure
	case linkRE.MatchString(text):
		return "", WebLink
	case !canon.IsStatement(text), cut && !taskRE.MatchString(text), !strings.ContainsAny(text, "$?") && personRE.MatchString(text) && !taskRE.MatchString(text):
		return "", NotStated
	case !balanced(text):
		return "", BrokenTeX
	}
	return text, ""
}

// balanced tells if the dollar signs of a text that are not escaped pair up, and if the braces inside the formulas pair up.
func balanced(text string) bool {
	dollars, depth := 0, 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '$':
			dollars++
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return dollars%2 == 0 && depth == 0
}

// languages maps the names of languages that the datasets give to their codes.
var languages = map[string]string{
	"english": "en", "en": "en", "spanish": "es", "español": "es", "es": "es", "russian": "ru", "ru": "ru",
	"chinese": "zh", "chinese (traditional)": "zh", "zh": "zh", "mongolian": "mn", "macedonian": "mk",
	"german": "de", "de": "de", "vietnamese": "vi", "vi": "vi", "português": "pt", "portuguese": "pt", "pt": "pt",
	"french": "fr", "fr": "fr", "arabic": "ar", "afrikaans": "af",
}

// stopWords are common words of the languages that Lang tells apart by their words.
var stopWords = map[string][]string{
	"en": {"the", "of", "and", "that", "is", "are", "be", "let", "find", "prove", "all", "with", "for", "in"},
	"es": {"el", "la", "los", "las", "de", "que", "es", "un", "una", "sea", "con", "para", "por", "y", "en"},
	"pt": {"o", "os", "as", "de", "que", "um", "uma", "seja", "com", "para", "por", "e", "não", "são"},
	"fr": {"le", "la", "les", "de", "des", "que", "est", "un", "une", "soit", "pour", "et", "dans", "sont"},
	"de": {"der", "die", "das", "und", "ist", "eine", "ein", "sei", "für", "mit", "dass", "von", "zu", "auf"},
	"vi": {"của", "và", "là", "các", "cho", "với", "một", "có", "số", "tìm", "chứng", "minh", "rằng", "trong"},
}

// Lang gives the language of a text, from the language that the row names and from the text. A row may name two languages, as "English; Russian", or none. The text decides between them: its script, then its common words. It gives "" when the language is not clear.
func Lang(text, named string) string {
	var want []string
	for _, n := range strings.FieldsFunc(strings.ToLower(named), func(r rune) bool { return r == ';' || r == ',' }) {
		if c := languages[strings.TrimSpace(n)]; c != "" {
			want = append(want, c)
		}
	}
	var latin, cyrillic, han, kana, hangul, arabic int
	for _, r := range stripMath(text) {
		switch {
		case unicode.Is(unicode.Latin, r):
			latin++
		case unicode.Is(unicode.Cyrillic, r):
			cyrillic++
		case unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			kana++
		case unicode.Is(unicode.Han, r):
			han++
		case unicode.Is(unicode.Hangul, r):
			hangul++
		case unicode.Is(unicode.Arabic, r):
			arabic++
		}
	}
	got := ""
	switch {
	case kana > 0:
		got = "ja"
	case hangul > latin/4 && hangul > 0:
		got = "ko"
	case han > latin/4 && han > 0:
		got = "zh"
	case arabic > latin/4 && arabic > 0:
		got = "ar"
	case cyrillic > latin/4 && cyrillic > 0:
		// The Cyrillic languages are not told apart by their script, so the row must name one.
		for _, w := range want {
			if w == "ru" || w == "mk" || w == "mn" {
				return w
			}
		}
		return ""
	default:
		got = byWords(text)
		if got == "" && len(want) == 1 && latinLang(want[0]) {
			// A text of mostly formulas has few words, so the language that the row names stands.
			return want[0]
		}
	}
	if got == "" {
		return ""
	}
	if len(want) == 0 || contains(want, got) || got == "en" {
		return got
	}
	return ""
}

// latinLang tells if a language is written in the Latin script.
func latinLang(c string) bool { _, ok := stopWords[c]; return ok || c == "af" }

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// byWords gives the Latin language whose common words are the most frequent in the text, when they are clearly more frequent than the words of the next language.
func byWords(text string) string {
	counts := map[string]int{}
	words := strings.FieldsFunc(strings.ToLower(stripMath(text)), func(r rune) bool { return !unicode.IsLetter(r) })
	for _, w := range words {
		for lang, list := range stopWords {
			if contains(list, w) {
				counts[lang]++
			}
		}
	}
	best, next := "", 0
	for _, lang := range []string{"en", "es", "pt", "fr", "de", "vi"} {
		switch c := counts[lang]; {
		case best == "" || c > counts[best]:
			next = counts[best]
			best = lang
		case c > next:
			next = c
		}
	}
	if counts[best] < 1 || counts[best] < 2*next {
		return ""
	}
	return best
}

// mathRE finds the formulas of a text.
var (
	// detailsRE is a folded block of a post, such as the original wording or a note of the poster.
	detailsRE = regexp.MustCompile(`(?is)<details>.*?</details>`)
	// hideNoteRE is the note at the end of the posts of some AoPS collections.
	hideNoteRE = regexp.MustCompile(`(?i)PS\.? You (?:should|can) use hide for answers\.?\s*(?:Collected \[here\]\([^)]*\)\.?)?`)
	// tagRE is an HTML tag of style, which the text does not need.
	tagRE = regexp.MustCompile(`(?i)</?(?:span|u|b|i|em|strong|font|center|sup|sub)(?:\s[^<>]*)?>`)
	brRE  = regexp.MustCompile(`(?i)<br\s*/?>`)
	// bbRE is a BBCode tag of style of AoPS.
	bbRE   = regexp.MustCompile(`(?i)\[/?(?:color|size|font|center|u)(?:=[^\]]*)?\]`)
	bbList = strings.NewReplacer("[list]", "\n", "[list=1]", "\n", "[list=a]", "\n", "[/list]", "\n", "[*]", "\n- ", "[i]", "*", "[/i]", "*", "[b]", "**", "[/b]", "**")
	// linkRE is a web link. A text with a link sends the reader to a page for a part of the problem or for its context.
	linkRE = regexp.MustCompile(`https?://|www\.[a-z]`)
)

// markup removes the HTML and the BBCode of a post that only style the text, the folded blocks and the notes of the poster.
func markup(text string) string {
	text = detailsRE.ReplaceAllString(text, "")
	text = hideNoteRE.ReplaceAllString(text, "")
	text = brRE.ReplaceAllString(text, "\n")
	text = tagRE.ReplaceAllString(text, "")
	text = bbRE.ReplaceAllString(text, "")
	return strings.TrimSpace(bbList.Replace(text))
}

var mathRE = regexp.MustCompile(`(?s)\$\$.*?\$\$|\$[^$]*\$`)

func stripMath(text string) string { return mathRE.ReplaceAllString(text, " ") }
