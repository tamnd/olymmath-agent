package canon

import (
	"regexp"
	"strings"
)

var (
	// underlineRE is a heading of a translated problems.ru row, such as "$\underline{\text { Kozhevnikov P. }}$.".
	underlineRE = regexp.MustCompile(`^\$\\underline\{\\text\s*\{[^}]*\}\}\$\.*$`)
	// bylineRE is a short line that names the author of a problem, such as "Bogdanov I.I.", "Sonkin $M$." or "Golovanov A.S. $^{2}$": it ends with an initial.
	bylineRE = regexp.MustCompile(`(?:^|[\s.$])\p{L}\$?\s?\.+(?:\s*\$\^\{\d+\}\$)?$`)
	// authorsRE is a line that starts with a word for authors, such as "Author: A. Golovanov." or "Authors: I. Bogdanov, D. von der Flaass: On a plane ...".
	authorsRE = regexp.MustCompile(`^(?:Authors?|Avoro|Автор\p{L}*)\s*:\s*([^:\n]{1,60})(?::\s+|$)`)
	// nameLeadRE is the name of the author at the start of the text, as in "Berlov S.L. On the arcs ...".
	nameLeadRE = regexp.MustCompile(`^\p{L}[\p{L}'’-]+\s+(?:\p{L}\.\s?){2}\s*(\p{Lu})`)
	// answerRE is a short answer at the end of a text, such as "Answer: 1:2.".
	answerRE = regexp.MustCompile(`\s+Answer\s*:[^\n]{1,40}$`)
	// markRE is the number of a problem at the start of a text, such as "11.4.", "54.", "Problem 1. (4 points)" or "18.6.3 ${ }^{\star}$".
	markRE = regexp.MustCompile(`^(?:Problem\s+)?(?:[0-9]{1,2}(?:\.[0-9]{1,2}){0,2}\.|[0-9]{1,2}(?:\.[0-9]{1,2}){1,2})\s+(?:\([0-9]+\s*points?\)\s*)?(?:\$\{\s*\}\^\{\\star\}\$\s*|\$\\rightarrow\$\s*)?([\p{Lu}$])`)
)

// meta tells if a line of the head of a text names its author or its topic and does not state the problem.
func meta(l string) bool {
	if underlineRE.MatchString(l) || authorsRE.MatchString(l) && !strings.Contains(authorsRE.ReplaceAllString(l, ""), " ") {
		return true
	}
	if strings.HasPrefix(strings.TrimLeft(l, "0123456789, "), "[") && len(strings.Fields(l)) <= 12 {
		return true
	}
	return len(strings.Fields(l)) <= 4 && len(l) <= 40 && !strings.ContainsAny(l, "?=") && bylineRE.MatchString(l)
}

// topic tells if a line can be the topic of a text, such as "Coloring": a few words with no formula and no sentence mark.
func topic(l string) bool {
	return len(strings.Fields(l)) <= 6 && !strings.ContainsAny(l, "$.?!=:")
}

// CleanHead removes the head of a seed row that the content match placed: the lines that name its author or its topic, such as "Bogdanov I.I." or "[ Two tangents drawn from one point ]", the number of the problem at the start and a short answer at the end. Some datasets translate these from the problems.ru page. A head with a topic line is removed only when a line of the head names the author or the topic for sure.
func CleanHead(text string) string {
	text = strings.TrimSpace(text)
	for range 3 {
		lines := strings.SplitN(text, "\n", 4)
		cut, rest := -1, ""
		for i, l := range lines {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if meta(l) && i < len(lines)-1 {
				cut, rest = i, ""
				continue
			}
			if m := authorsRE.FindStringIndex(l); m != nil {
				// The statement goes on after the names of the authors.
				cut, rest = i, l[m[1]:]
				break
			}
			if !topic(l) {
				break
			}
		}
		if cut < 0 {
			break
		}
		text = strings.TrimSpace(rest + "\n" + strings.Join(lines[cut+1:], "\n"))
	}
	if m := nameLeadRE.FindStringSubmatchIndex(text); m != nil {
		text = text[m[2]:]
	}
	if m := markRE.FindStringSubmatchIndex(text); m != nil {
		text = text[m[2]:]
	}
	return answerRE.ReplaceAllString(text, "")
}
