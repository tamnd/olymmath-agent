package cli

import (
	"slices"
	"testing"

	"github.com/tamnd/olymmath-agent/official"
)

func TestHrArchive(t *testing.T) {
	page := `<h2>Školska/gradska natjecanja</h2>
<table><tr><th>Godina</th><th>OŠ</th><th>SŠ A</th><th>SŠ B</th></tr>
<tr><td>2024.</td><td><a href="https://x.hr/wp-content/uploads/2024/01/os.pdf">Zadaci</a></td><td><a href="https://x.hr/wp-content/uploads/2024/01/a.pdf">Zadaci</a></td><td>—</td></tr></table>
<h2>Državna natjecanja</h2>
<table><tr><td>2019.</td><td>—</td><td><a href="https://x.hr/b&amp;c.pdf">Zadaci</a></td><td><a href="https://x.hr/b.pdf">Zadaci</a></td></tr></table>
<h3>Slika kao izazov</h3><table><tr><td>2020.</td><td><a href="https://x.hr/no.pdf">x</a></td></tr></table>`
	got := hrArchive(page)
	want := []hrPaper{
		{kind: "hr-city", year: 2024, variant: "o", url: "https://x.hr/wp-content/uploads/2024/01/os.pdf"},
		{kind: "hr-city", year: 2024, variant: "a", url: "https://x.hr/wp-content/uploads/2024/01/a.pdf"},
		{kind: "hr-state", year: 2019, variant: "a", url: "https://x.hr/b&c.pdf"},
		{kind: "hr-state", year: 2019, variant: "b", url: "https://x.hr/b.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("hrArchive gives\n%+v\nwant\n%+v", got, want)
	}
}

func TestHrRounds(t *testing.T) {
	page := `<a href="https://x.hr/wp-content/uploads/2026/08/HMOK-2022-prvo_kolo-zadaci.pdf">Zadaci</a>
<a href="https://x.hr/wp-content/uploads/2026/08/HMOK-2022-prvo_kolo-rjesenja.pdf">Rješenja</a>
<a href="https://x.hr/wp-content/uploads/2026/08/MEDO-2024_2025-drugo_kolo-zadaci.pdf">Zadaci</a>
<a href="https://x.hr/wp-content/uploads/2026/08/HMOK-2022-prvo_kolo-zadaci.pdf">Zadaci</a>`
	got := hrRounds(page)
	if len(got) != 2 || got[0].kind != "hmok" || got[0].year != 2022 || got[0].variant != "prvo" || got[1].kind != "medo" || got[1].year != 2024 || got[1].variant != "drugo" {
		t.Errorf("hrRounds gives %+v", got)
	}
}

func TestHrPlace(t *testing.T) {
	pr := func(code, section string, n int) official.HrProblem {
		return official.HrProblem{DemoProblem: official.DemoProblem{Code: code}, N: n, Section: section}
	}
	for _, c := range []struct {
		paper hrPaper
		probs []official.HrProblem
		want  []string
	}{
		{hrPaper{kind: "hr-county", year: 2023, variant: "b"}, []official.HrProblem{pr("1", "3", 1), pr("7", "3", 7), pr("1", "", 1)}, []string{"hr-county/2023/b31", "hr-county/2023/b37", ""}},
		{hrPaper{kind: "hr-mo", year: 2024, variant: "p"}, []official.HrProblem{pr("1", "Prvi dan", 1), pr("1", "Drugi dan", 1), pr("2", "Završni test za izbor IMO ekipe", 2), pr("1", "Završni test za izbor MEMO ekipe", 1)}, []string{"hr-mo/2024/a1", "hr-mo/2024/b1", "hr-tst/2024/i1", "hr-tst/2024/m1"}},
		{hrPaper{kind: "hr-jmo", year: 2021, variant: "p"}, []official.HrProblem{pr("1", "", 1), pr("2", "", 2)}, []string{"hr-jmo/2021/p1", "hr-jmo/2021/p2"}},
		{hrPaper{kind: "hmok", year: 2022, variant: "prvo"}, []official.HrProblem{pr("10", "4", 10), pr("1", "5", 1)}, []string{"hr-cadets-r1/2022/o410", "hr-cadets-r1/2022/o501"}},
		{hrPaper{kind: "hmok", year: 2024, variant: "drugo"}, []official.HrProblem{pr("5", "B", 5)}, []string{"hr-cadets/2024/b5"}},
		{hrPaper{kind: "medo", year: 2025, variant: "treće"}, []official.HrProblem{pr("K-3.8", "2025", 8)}, []string{"hr-medo/2025/k38"}},
	} {
		ats, names := hrPlace(c.paper, c.probs)
		if !slices.Equal(ats, c.want) || len(names) != len(ats) {
			t.Errorf("hrPlace(%+v) = %v, want %v", c.paper, ats, c.want)
		}
	}
}

func TestHrFix(t *testing.T) {
	p := official.HrProblem{DemoProblem: official.DemoProblem{Text: "(10 bodova) Koliko ima podskupova?"}}
	hrHarvest.fix(&p)
	if p.Text != "Koliko ima podskupova?" {
		t.Errorf("hrHarvest.fix gives %q", p.Text)
	}
}
