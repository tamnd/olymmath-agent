package mathse

import (
	"strings"
	"testing"
)

func TestTitleKey(t *testing.T) {
	for title, want := range map[string]Key{
		"IMO 1987 - function such that $f(f(n))=n+1987$": {"imo/1987", ""},
		"Vieta jumping, IMO 1988 Problem 6":              {"imo/1988", "p6"},
		"IMO 2009 Q1: a question":                        {"imo/2009", "p1"},
		"A problem from the IMO Shortlist 2007 (N4)":     {"isl/2007", "n4"},
		"ISL 2015 A1 inequality":                         {"isl/2015", "a1"},
		"Putnam 1990 A3":                                 {"putnam/1990", "a3"},
		"Putnam 2003 B-4 rational roots":                 {"putnam/2003", "b4"},
		"USAMO 1999 Problem 2":                           {"usamo/1999", "p2"},
		"USAJMO 2014 #3":                                 {"usajmo/2014", "p3"},
		"Inequality from the 2005 IMO":                   {"imo/2005", ""},
	} {
		got, ok := TitleKey(title)
		if !ok || got != want {
			t.Errorf("%q: %+v %v", title, got, ok)
		}
	}
	for _, title := range []string{"An inequality with three variables", "A generalization of IMO 1977 problem 2", "An IMO 1984 longlisted problem"} {
		if _, ok := TitleKey(title); ok {
			t.Errorf("%q gave a key", title)
		}
	}
}

func TestParse(t *testing.T) {
	p, err := Parse([]byte(`{"items":[{"answer_id":5,"question_id":3,"body_markdown":"Let $x &lt; y$.\r\nThen","owner":{"display_name":"A &amp; B"},"is_accepted":true}],"has_more":false,"quota_remaining":9}`))
	if err != nil || len(p.Items) != 1 || p.Items[0].Body != "Let $x < y$.\nThen" || p.Items[0].Owner.Name != "A & B" || p.Quota != 9 {
		t.Errorf("%+v %v", p, err)
	}
	if _, err := Parse([]byte(`{"error_id":502,"error_message":"too many requests"}`)); err == nil {
		t.Error("an error of the API was not returned")
	}
}

func TestContains(t *testing.T) {
	problem := "Prove that for every positive integer n the number n squared plus n is even."
	question := "I am stuck. " + problem + " My attempt: I tried induction on n but it did not work."
	if c := Contains(problem, question); c != 1 {
		t.Errorf("contains %v", c)
	}
	if c := Contains(problem, strings.Repeat("other words here ", 5)); c != 0 {
		t.Errorf("contains %v", c)
	}
}

func TestURLs(t *testing.T) {
	if u := AnswersURL([]int{1, 2}, 1); !strings.HasPrefix(u, API+"/questions/1;2/answers?") {
		t.Error(u)
	}
	if u := SearchURL("IMO", 2); !strings.Contains(u, "tagged=contest-math") || !strings.Contains(u, "page=2") {
		t.Error(u)
	}
}

func TestIsSolution(t *testing.T) {
	long := strings.Repeat("Let $x$ be a real number. ", 15)
	if !IsSolution(long) {
		t.Error("a long answer was refused")
	}
	for _, b := range []string{"Short.", long + "Your proof is right.", "**Hint:** " + long, long + "This is verbatim from imomath."} {
		if IsSolution(b) {
			t.Errorf("accepted %.40q", b)
		}
	}
}
