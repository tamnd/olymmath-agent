package loose

import (
	"regexp"
	"slices"
	"strings"
)

// The rule of the words (spec 06.8). Two copies of one problem from two sources often write each formula in another way: "$A,B,C$" and "$A$, $B$, $C$", a system in a bare align or in "$$", "\sin x" and "\sin{x}". Signals E, T and F all read the formulas, so for such a pair all three are low, and the pair is not even compared. But the words outside the formulas are the same, and so are the letters and the digits inside them.
const (
	// WordNear is the lowest TF-IDF cosine of the words of two texts that are compared by this rule.
	WordNear = 0.7
	// WordSame is the lowest Jaccard index of the words of two texts that are the same by this rule.
	WordSame = 0.8
	// FormulaSame is the lowest Jaccard index of the letters, digits and commands of the formulas.
	FormulaSame = 0.75
	// MinWords is the fewest words of each text. A short text such as "Find all functions $f$ such that ... for all reals $x, y$" has the words of many problems.
	MinWords = 10
	// ShortWords is the number of words below which the formulas must agree in their order too: the Jaccard index of the pairs of adjacent tokens of the formulas, with the signs, must be OrderSame or more. Two functional equations in the same words often have the same letters in another order.
	ShortWords = 20
	OrderSame  = 0.7
)

var (
	// envRE is a formula that is not in dollars: an environment such as a bare \begin{align*}, or \[ \] and \( \).
	envRE  = regexp.MustCompile(`(?s)\\begin\{[a-zA-Z]+\*?\}.*?\\end\{[a-zA-Z]+\*?\}|\\\[.*?\\\]|\\\(.*?\\\)`)
	cmdRE  = regexp.MustCompile(`\\[A-Za-z]+`)
	wordRE = regexp.MustCompile(`[a-z]+`)
	// symRE is a token of a formula: a command or one letter or digit. opRE is that or a sign.
	symRE    = regexp.MustCompile(`\\[A-Za-z]+|[A-Za-z0-9]`)
	digitsRE = regexp.MustCompile(`[0-9]+`)
	opRE     = regexp.MustCompile(`\\[A-Za-z]+|[A-Za-z0-9]|[-+=<>()|*/!]`)
	// layout are the commands that only set the look of a formula.
	layout = map[string]bool{}
)

func init() {
	for _, c := range strings.Fields(`left right big Big bigg Bigg bigl bigr Bigl Bigr displaystyle text textbf textit textrm mathrm mathbf mathit operatorname mbox begin end
		frac dfrac tfrac binom dbinom tbinom quad qquad hspace vspace limits nolimits cdot cdots ldots dots`) {
		layout[`\`+c] = true
	}
}

// shape is the words of a text outside its formulas, their number and their text, the tokens of its formulas and the pairs of adjacent tokens with the signs, and its numbers.
type shape struct {
	words    map[string]int
	n        int
	text     string
	formulas map[string]int
	order    map[string]int
	numbers  string
}

// shapeOf gives the shape of a match form. The words and the formulas are those of the stem, and the numbers are those of the whole form, so two problems with the same stem and other choices differ.
func shapeOf(form string) shape {
	st := stem(form)
	var fs []string
	for _, re := range []*regexp.Regexp{mathRE, envRE} {
		st = re.ReplaceAllStringFunc(st, func(m string) string {
			fs = append(fs, m)
			return " "
		})
	}
	s := shape{words: map[string]int{}, formulas: map[string]int{}, order: map[string]int{}}
	ws := wordRE.FindAllString(strings.ToLower(cmdRE.ReplaceAllString(st, " ")), -1)
	for _, w := range ws {
		s.words[w]++
	}
	s.n, s.text = len(ws), strings.Join(ws, " ")
	prev := ""
	for _, f := range fs {
		for _, t := range symRE.FindAllString(f, -1) {
			if !layout[t] {
				s.formulas[t]++
			}
		}
		for _, t := range opRE.FindAllString(f, -1) {
			if !layout[t] {
				s.order[prev+" "+t]++
				prev = t
			}
		}
	}
	nums := map[string]bool{}
	for _, d := range digitsRE.FindAllString(form, -1) {
		if d = strings.TrimLeft(d, "0"); d == "" {
			d = "0"
		}
		nums[d] = true
	}
	keys := make([]string, 0, len(nums))
	for d := range nums {
		keys = append(keys, d)
	}
	slices.Sort(keys)
	s.numbers = strings.Join(keys, " ")
	return s
}

// bagJaccard is the Jaccard index of two multisets.
func bagJaccard(a, b map[string]int) float64 {
	inter, union := 0, 0
	for k, x := range a {
		y := b[k]
		inter += min(x, y)
		union += max(x, y)
	}
	for k, y := range b {
		if _, ok := a[k]; !ok {
			union += y
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
