package seed

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/tamnd/olymmath-agent/content"
)

// AliasFile is the path of the alias table in the content repository. Each line is a label, a tab and a competition code. A line that starts with "#" is a comment.
const AliasFile = "registers/competition-aliases.tsv"

// Aliases resolves a competition label to a register code.
type Aliases struct {
	m map[string]string
}

var ordinalRE = regexp.MustCompile(`\b[0-9]+(st|nd|rd|th)\b`)

var camelRE = regexp.MustCompile(`([a-z])([A-Z])`)

// expand gives the long form of the usual short words of competition names. Both sides of a lookup go through Norm, thus "Balkan MO" and "Balkan Mathematical Olympiad" give the same key.
var expand = map[string]string{
	"mo":          "mathematical olympiad",
	"tst":         "team selection test",
	"tsts":        "team selection test",
	"sl":          "shortlist",
	"math":        "mathematical",
	"maths":       "mathematical",
	"mathematics": "mathematical",
	"olympiads":   "olympiad",
	"contests":    "contest",
	"&":           "and",
}

var (
	letterDigitRE = regexp.MustCompile(`([a-z])([0-9])`)
	digitLetterRE = regexp.MustCompile(`([0-9])([a-z])`)
	yearWordRE    = regexp.MustCompile(`^(19|20)[0-9]{2}$`)
	shortNumRE    = regexp.MustCompile(`^[0-9]{1,2}$`)
	romanRE       = regexp.MustCompile(`^[ivxl]+$`)
)

// Norm gives the key of a label: lower case, words of a CamelCase name apart, no years, no year ranges, no ordinals, no edition number at the start, no punctuation, the long form of short words and single spaces. "51st Ukrainian National Mathematical Olympiad, 2015" gives "ukrainian national mathematical olympiad", and "NewZealand_MO" gives "new zealand mathematical olympiad".
func Norm(s string) string {
	s = camelRE.ReplaceAllString(s, "$1 $2")
	s = strings.ToLower(s)
	s = ordinalRE.ReplaceAllString(s, " ")
	s = letterDigitRE.ReplaceAllString(s, "$1 $2")
	s = digitLetterRE.ReplaceAllString(s, "$1 $2")
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, s)
	var words []string
	afterYear := false
	for _, w := range strings.Fields(s) {
		switch {
		case yearWordRE.MatchString(w):
			afterYear = true
			continue
		case afterYear && shortNumRE.MatchString(w):
			// The end of a range such as "1997-98".
			afterYear = false
			continue
		}
		afterYear = false
		if x, ok := expand[w]; ok {
			w = x
		}
		words = append(words, w)
	}
	// An edition number or a roman numeral at the start, as in "53. Bulgarian Mathematical Olympiad" or "IV Soros Olympiad".
	if len(words) > 1 && (romanRE.MatchString(words[0]) || shortNumRE.MatchString(words[0]) || strings.Trim(words[0], "0123456789") == "") {
		words = words[1:]
	}
	if len(words) > 0 && words[0] == "the" {
		words = words[1:]
	}
	return strings.Join(words, " ")
}

// LoadAliases makes the alias table from the register (each code, short name, name and other name) and from the alias file. A row of the alias file wins over the register.
func LoadAliases(root string, corpus *content.Corpus) (*Aliases, error) {
	a := &Aliases{m: map[string]string{}}
	for _, c := range corpus.Competitions {
		for _, l := range []string{c.Code, c.Short, c.Name} {
			a.add(l, c.Code)
		}
		for _, l := range c.Names {
			a.add(l, c.Code)
		}
	}
	f, err := os.Open(root + "/" + AliasFile) // #nosec G304 -- the path is inside the content root.
	if os.IsNotExist(err) {
		return a, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		label, code, ok := strings.Cut(line, "\t")
		code = strings.TrimSpace(code)
		if !ok || code == "" {
			return nil, fmt.Errorf("%s:%d: a line needs a label, a tab and a code", AliasFile, n)
		}
		if code != "-" && corpus.Competition(code) == nil {
			return nil, fmt.Errorf("%s:%d: %q is not in the competition register", AliasFile, n, code)
		}
		a.m[Norm(label)] = code
	}
	return a, sc.Err()
}

func (a *Aliases) add(label, code string) {
	if k := Norm(label); k != "" {
		if _, ok := a.m[k]; !ok {
			a.m[k] = code
		}
	}
}

// Code returns the code of the first label that resolves. The code "-" marks a label that is known not to be a competition of the register, for example a textbook set. Such a label gives "-".
func (a *Aliases) Code(labels []string) string {
	for _, l := range labels {
		if c, ok := a.m[Norm(l)]; ok {
			return c
		}
	}
	return ""
}

// Len is the number of keys.
func (a *Aliases) Len() int { return len(a.m) }

var posRE = regexp.MustCompile(`^(?:problem|prob|p|q|#)?\s*([a-z]?)\s*([0-9]{1,4})$`)

// Position makes a position from a problem label of a dataset: "3", "P3" and "Problem 3" give "p3", and "A1" and "A-1" give "a1". A number of three or four digits needs a letter, for example the KöMaL sign B3802, because a long number alone is often a year. It returns "" for a label it cannot read.
func Position(label string) string {
	m := posRE.FindStringSubmatch(strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(label))))
	if m == nil {
		return ""
	}
	n, _ := strconv.Atoi(m[2])
	if n == 0 || (m[1] == "" && len(m[2]) > 2) {
		return ""
	}
	if m[1] != "" {
		return m[1] + strconv.Itoa(n)
	}
	return "p" + strconv.Itoa(n)
}
