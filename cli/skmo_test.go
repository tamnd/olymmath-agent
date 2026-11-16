package cli

import (
	"slices"
	"testing"

	"github.com/tamnd/olymmath-agent/official"
)

func TestSkmoTable(t *testing.T) {
	page := `<table class="terminy">
  <tr>
    <th rowspan="2">kategória \ kolo</th>
    <th colspan="3">domáce</th>
    <th colspan="2">školské</th>
    <th colspan="2">okresné</th>
    <th colspan="2">krajské</th>
    <th colspan="3">celoštátne</th>
  </tr>
  <tr><th>zadania</th><th>návody</th><th>riešenia</th><th>zadania</th><th>riešenia</th><th>zadania</th><th>riešenia</th><th>zadania</th><th>riešenia</th><th>zadania</th><th>riešenia</th><th>poradie</th></tr>
  <tr>
    <th>A</th>
    <td class="medzil"><a href="dokument.php?id=5423">pdf</a> [<span class="tooltip">?<span class="tooltiptext">verzia v slovenčine</span></span>]<br /><a href="dokument.php?id=5815">pdf</a> [<span class="tooltip">?<span class="tooltiptext">verzia v maďarčine</span></span>]</td>
    <td class="medzi"><a href="dokument.php?id=5811">pdf</a></td>
    <td class="medzir"><a href="dokument.php?id=5825">pdf</a></td>
    <td class="medzil"><a href="dokument.php?id=5826">pdf</a></td>
    <td class="medzir"></td>
    <td colspan="2" class="nenitermin"></td>
    <td class="medzil"><a href="dokument.php?id=5835">pdf</a></td>
    <td class="medzir"></td>
    <td class="medzil"><a href="dokument.php?id=5863">pdf</a></td>
    <td class="medzi"></td>
    <td class="medzir"><a href="dokument.php?id=6054">pdf</a></td>
  </tr>
  <tr>
    <th>Z9</th>
    <td class="medzil"><a href="dokument.php?id=5429">pdf</a></td>
    <td class="medzi"></td>
    <td class="medzir"></td>
    <td colspan="2" class="nenitermin"></td>
    <td class="medzil"><a href="dokument.php?id=5845">pdf</a></td>
    <td class="medzir"></td>
    <td colspan="5" class="nenitermin"></td>
  </tr>
  <tr>
    <th>Z4</th>
    <td class="medzil"><a href="dokument.php?id=9">pdf</a></td>
  </tr>
</table>`
	got := skmoTable(page, 75)
	u := func(id string) string { return "https://skmo.sk/dokument.php?id=" + id }
	want := []hrPaper{
		{kind: "A", variant: "home", url: u("5423"), year: 2026},
		{kind: "A", variant: "school", url: u("5826"), year: 2026},
		{kind: "A", variant: "regional", url: u("5835"), year: 2026},
		{kind: "A", variant: "national", url: u("5863"), year: 2026},
		{kind: "Z9", variant: "home", url: u("5429"), year: 2026},
		{kind: "Z9", variant: "district", url: u("5845"), year: 2026},
	}
	if !slices.Equal(got, want) {
		t.Errorf("skmoTable gives\n%+v\nwant\n%+v", got, want)
	}
}

func TestSkmoPlace(t *testing.T) {
	probs := []official.HrProblem{{N: 1}, {N: 2}}
	for _, c := range []struct {
		kind, variant string
		want          string
	}{
		{"A", "home", "cz-sk-mo-r1/2026/ha2"},
		{"C", "school", "cz-sk-mo-r1/2026/sc2"},
		{"B", "regional", "cz-sk-mo-r2/2026/b2"},
		{"A", "national", "cz-sk-mo/2026/p2"},
		{"Z7", "home", "cz-sk-mo-z/2026/h72"},
		{"Z5", "district", "cz-sk-mo-z/2026/d52"},
		{"Z9", "regional", "cz-sk-mo-z/2026/r92"},
		{"B", "national", ""},
	} {
		ats, names := skmoPlace(hrPaper{kind: c.kind, variant: c.variant, year: 2026}, probs)
		if len(ats) != 2 || ats[1] != c.want || len(names) != 2 {
			t.Errorf("skmoPlace(%s, %s) = %v, want %s", c.kind, c.variant, ats, c.want)
		}
	}
	if _, names := skmoPlace(hrPaper{kind: "A", variant: "home", year: 2026}, probs); names[0] != "skmo.sk, Matematická olympiáda 2025/2026, domáce kolo, kategória A" {
		t.Errorf("skmoPlace names %q", names[0])
	}
}

func TestSkmoText(t *testing.T) {
	for in, want := range map[string]string{
		"Určte štvrtý koreň tejto rovnice. (P. Černek)":                         "Určte štvrtý koreň tejto rovnice.",
		"Pre ktoré n sa tak môže stať? (Vojtech Bálint, Michal Rolínek)":        "Pre ktoré n sa tak môže stať?",
		"Dokážte, že priamky AK, BL prechádzajú bodom. (ABC)":                   "Dokážte, že priamky AK, BL prechádzajú bodom. (ABC)",
		"Ak je prienikom všetkých daných troj- uholníkov bod, určte $n$.":       "Ak je prienikom všetkých daných trojuholníkov bod, určte $n$.",
		"Nájdite čísla s vlastnosťami: • Na mieste desiatok je 5. • Je menšie.": "Nájdite čísla s vlastnosťami:\n\n- Na mieste desiatok je 5.\n- Je menšie.",
	} {
		if got := skmoText(in); got != want {
			t.Errorf("skmoText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSkmoRules(t *testing.T) {
	if !skmoRules.Not.MatchString("4 360. Aké dve štvorciferné čísla to mohli byť?") || skmoRules.Not.MatchString("4 Na tabuli je 360 čísel.") {
		t.Error("skmoRules.Not")
	}
	if !skmoRules.Stop.MatchString("Skolské kolo MO kategórie A sa koná v utorok") || skmoRules.Stop.MatchString("Juro mal tri zhodné obdĺžniky.") {
		t.Error("skmoRules.Stop")
	}
	if !skmoDropRE.MatchString("(Jaroslav Svrček)") || !skmoDropRE.MatchString("(P. Černek)") || skmoDropRE.MatchString("(pozri obrázok)") {
		t.Error("skmoDropRE")
	}
	if !skmoLossy.MatchString("CSYIOV+Cambria") || skmoLossy.MatchString("csr10") {
		t.Error("skmoLossy")
	}
}
