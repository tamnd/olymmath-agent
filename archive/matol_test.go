package archive

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// matolFixture reads a page of testdata/matol: the main part of a page of the site as captured on 2026-11-17.
func matolFixture(t *testing.T, name string) MatolPage {
	t.Helper()
	b, err := os.ReadFile("testdata/matol/" + name + ".html")
	if err != nil {
		t.Fatal(err)
	}
	return ParseMatol(b)
}

func TestParseMatolNode(t *testing.T) {
	p := matolFixture(t, "nodes-11")
	if p.Leaf != "Районный этап" || len(p.Crumbs) != 3 || p.Crumbs[2].Path != "/nodes/10" {
		t.Fatalf("leaf %q crumbs %v", p.Leaf, p.Crumbs)
	}
	if !slices.Contains(p.Links, "/nodes/522") || !slices.Contains(p.Links, "/nodes/26") {
		t.Fatalf("links %v", p.Links)
	}
	if len(p.Problems) != 0 {
		t.Fatalf("%d problems on a node page", len(p.Problems))
	}
}

func TestParseMatolPaper(t *testing.T) {
	p := matolFixture(t, "olympiads-1137")
	if len(p.Problems) != 3 {
		t.Fatalf("%d problems", len(p.Problems))
	}
	for i, pr := range p.Problems {
		if pr.N != i+1 || pr.Comment != strconv.Itoa(5705+i) {
			t.Errorf("problem %d: number %d, comment %s", i, pr.N, pr.Comment)
		}
		if strings.Contains(pr.Body, "<font") {
			t.Errorf("problem %d keeps the authors: %q", pr.N, pr.Body)
		}
	}
	if len(p.Links) != 0 {
		t.Fatalf("links %v on a paper page", p.Links)
	}
	f := matolFixture(t, "olympiads-1153")
	if len(f.Problems) != 6 || !slices.Equal(f.Problems[0].Authors, []string{"Сатылханов К."}) {
		t.Fatalf("%d problems, authors %v", len(f.Problems), f.Problems[0].Authors)
	}
}

func TestMatolPlace(t *testing.T) {
	for _, c := range []struct {
		page string
		n    int
		want string
	}{
		// The district stage of 2024/25 was written in the autumn of 2024.
		{"olympiads-1137", 2, "kz-mo-district/2024/p92"},
		{"olympiads-950", 1, "kz-mo-district/2020/p101"},
		// The junior olympiad numbers its stages d, r and f, all in the edition of the last year.
		{"olympiads-1180", 30, "kz-junior/2023/d730"},
		{"olympiads-1005", 8, "kz-smagulov/2020/a68"},
		{"olympiads-1013", 5, "kz-smagulov/2022/t5"},
		// The final stage numbers grade 11 as the national olympiad, and the grades 9 and 10 after the grade.
		{"olympiads-1154", 3, "kz-mo/2025/p3"},
		{"olympiads-1153", 1, "kz-mo/2025/p101"},
		{"olympiads-1220", 8, "kz-zhautykov-city/2026/p68"},
		// The Almaty olympiad: the heading gives the year up to 2023, and from the 18th olympiad the sitting is the year before the name.
		{"olympiads-1046", 4, "kz-almaty/2023/p4"},
		{"olympiads-1250", 1, "kz-almaty/2026/p1"},
		// A paper for the grades 5 and 6 takes the first.
		{"olympiads-411", 20, "kz-al-farabi/2016/p520"},
	} {
		code, year, number, why := MatolPlace(matolFixture(t, c.page), c.n)
		if why != "" {
			t.Errorf("%s %d: %s", c.page, c.n, why)
			continue
		}
		if got := code + "/" + strconv.Itoa(year) + "/" + number; got != c.want {
			t.Errorf("%s %d: %s, want %s", c.page, c.n, got, c.want)
		}
	}
	for page, want := range map[string]string{
		"olympiads-1026": "the test round of the final stage of the junior olympiad",
		"olympiads-416":  "a paper of physics",
		"olympiads-1099": "the second sitting of 2023",
	} {
		if _, _, _, why := MatolPlace(matolFixture(t, page), 1); why != want {
			t.Errorf("%s: %q, want %q", page, why, want)
		}
	}
}

func TestMatolText(t *testing.T) {
	problem := func(page string, n int) MatolProblem {
		for _, pr := range matolFixture(t, page).Problems {
			if pr.N == n {
				return pr
			}
		}
		t.Fatalf("%s has no problem %d", page, n)
		return MatolProblem{}
	}
	s, why := MatolText(problem("olympiads-1153", 1))
	if why != "" || s != "Найдите все четверки натуральных чисел $(a,b,c,d)$ таких, что $2^a+3^b=5^c\\cdot d$ и $2^b+3^a=5^d\\cdot c$." {
		t.Errorf("%q %q", s, why)
	}
	if s, why := MatolText(problem("olympiads-1013", 8)); why != "" || !strings.Contains(s, "*") || strings.Contains(s, "<") {
		t.Errorf("italic: %q %q", s, why)
	}
	if _, why := MatolText(problem("olympiads-1005", 8)); why != "a figure" {
		t.Errorf("figure: %q", why)
	}
	if _, why := MatolText(problem("olympiads-950", 1)); why != "markup that does not convert" {
		t.Errorf("strike: %q", why)
	}
	if _, why := MatolText(MatolProblem{Body: "&nbsp;См. рис.<br/>"}); why != "a text too short to be a statement" {
		t.Errorf("short: %q", why)
	}
}
