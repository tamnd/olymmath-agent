package archive

import (
	"strings"
	"testing"
)

const prNodePage = `<a class="componentboxheaderlink" href="/view_by_source_new.php?parent=204563">
	Олимпиады</a>
 &gt;&gt; <a class="componentboxheaderlink" href="/view_by_source_new.php?parent=100">
	Олимпиада города</a>
	</div>
Годы:<ul class="componentboxlist">
	<li/>
	<a class="componentboxlink" href="/view_by_source_new.php?parent=101">
		2024 год	</a>
	(12 задач)
	<li/>
	<a class="componentboxlink" href="/view_by_source_new.php?parent=102">
		11 класс. Второй день	</a>
	(5 задач)
</ul>
<p/>Страница:  1  <a class="componentboxlink" href="/view_by_source_new.php?parent=100&start=5">2</a> [Всего задач: 7]`

func prItem(id, number, body string) string {
	return `<tr><td class="problemsmallnumber"><nobr><h3 style="display:inline">
					Задача
					<a class="componentboxlink" href="/view_problem_details_new.php?id=` + id + `">
						` + id + `</a>
					` + number + `				</h3></nobr></td>
		<td class="problemsmallsubject"><table><tr><td><a class="componentboxlink" href="/view_by_subject_new.php?parent=849">Тема</a></td></tr></table></td>
		<td class="problemsmalldifficulty">
<nobr>Сложность: 3</nobr><br/><nobr>Классы: 8,9</nobr><br/>
		</td>
	</tr>
</table>
<div class="catalogueproblemauthorold"><p/>Автор: <a href="/view_by_author.php?author=1">Иванов И.</a></div>
` + body + `<table class="problemfootertable"><tr><td><a href="/view_problem_details_new.php?id=` + id + `">Решение</a></td></tr></table>`
}

func TestPRNodes(t *testing.T) {
	ns := PRNodes(prNodePage)
	if len(ns) != 2 || ns[0] != (PRNode{ID: 101, Name: "2024 год", Count: 12}) || ns[1].Name != "11 класс. Второй день" {
		t.Fatalf("nodes %+v", ns)
	}
	if got := PRName(prNodePage); got != "Олимпиада города" {
		t.Fatalf("name %q", got)
	}
	if got := PRTotal(prNodePage); got != 7 {
		t.Fatalf("total %d", got)
	}
	if y, ok := PRYear(ns[0].Name); !ok || y != 2024 {
		t.Fatalf("year %d %v", y, ok)
	}
	if y, ok := PRYear("XV Олимпиада по геометрии имени И.Ф. Шарыгина (2019 г.)"); !ok || y != 2019 {
		t.Errorf("a numbered year gives %d, %v", y, ok)
	}
	if _, ok := PRYear("2 турнир (1980/1981 год)"); ok {
		t.Error("a year that spans two years is read")
	}
	if _, ok := PRYear("Отборочный тур"); ok {
		t.Fatal("a tour is not a year")
	}
	for name, want := range map[string]int{"44 (2021), математика": 2021, "30 (2007), математические игры": 0, "27 (2004)": 0} {
		if y, ok := PRYear(name); y != want || ok != (want != 0) {
			t.Errorf("%s: %d %v", name, y, ok)
		}
	}
	for name, want := range map[string][2]int{"46 турнир (2024/2025 год)": {2024, 2025}, "1 турнир (1980 год)": {1980, 1980}, "1992-1993": {1992, 1993}, "2 турнир (1980/1982 год)": {0, 0}, "2025 год": {0, 0}} {
		first, last, ok := PRSpan(name)
		if ok != (want[0] != 0) || (ok && (first != want[0] || last != want[1])) {
			t.Errorf("%s: %d %d %v", name, first, last, ok)
		}
	}
	if !PRFinal("Заключительный этап") || PRFinal("Региональный этап") {
		t.Error("the final round is not told from the regional round")
	}
	for name, want := range map[string]string{
		"осенний тур, базовый вариант, 8-9 класс":            "fall jo",
		"весенний тур, сложный вариант, 10-11 класс":         "spring sa",
		"осенний тур, тренировочный вариант, 7-8 класс":      "fall jo",
		"весенний тур, основной вариант, 9-10 класс":         "spring sa",
		"весенний тур, подготовительный вариант, 9-10 класс": "spring so",
		"весенний тур, вариант для москвичей, 7-8 класс":     "",
		"осенний тур, 7-8 класс":                             "",
		"устный тур":                                         "",
	} {
		tp, ok := PRTourPaper(name)
		got := ""
		if ok {
			got = "fall " + tp.Prefix()
			if tp.Spring {
				got = "spring " + tp.Prefix()
			}
		}
		if got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	for name, want := range map[string][3]int{"8 класс": {8, 0, 1}, "10 Класс": {10, 0, 1}, "11 класс. Первый день": {11, 1, 1}, "11 класс. Второй день": {11, 2, 1}, "Устный тур": {0, 0, 0}} {
		g, d, ok := PRPaper(name)
		if g != want[0] || d != want[1] || ok != (want[2] == 1) {
			t.Errorf("%s: %d %d %v", name, g, d, ok)
		}
	}
	if got := PRNodeURL(101, 0); got != "https://problems.ru/view_by_source_new.php?parent=101" {
		t.Errorf("url %s", got)
	}
	if got := PRNodeURL(101, 5); got != "https://problems.ru/view_by_source_new.php?parent=101&start=5" {
		t.Errorf("url %s", got)
	}
	if got := PRProblemURL(67463); got != "https://problems.ru/view_problem_details_new.php?id=67463" {
		t.Errorf("url %s", got)
	}
}

func TestParsePRListing(t *testing.T) {
	page := prItem("501", "&nbsp;(#1)", `<p/>Докажите, что $n^2 \geqslant n$ для числа <I>n</I>.`) +
		prItem("502", "&nbsp;(#2)", `<p/>Найдите <I>x</I><sup>2</sup> на рисунке.<div align="center"><img src="/show_document.php?id=1"></div>`) +
		prItem("503", "", `<p/>Задача без номера.<br/>а) первая часть; б) вторая.`) +
		prItem("504", "&nbsp;(#2010.10.4)", `<p/>Задача с полным номером.`) +
		prItem("505", "&nbsp;(#8.7)", `<p/>Задача с номером класса.`)
	ps := ParsePRListing(page)
	if len(ps) != 5 {
		t.Fatalf("problems %+v", ps)
	}
	if ps[0].ID != 501 || ps[0].Number != 1 || ps[0].Err != nil || ps[0].Text != `Докажите, что $n^2 \geqslant n$ для числа $n$.` {
		t.Errorf("first %+v", ps[0])
	}
	if ps[1].Number != 2 || ps[1].Err != nil || !ps[1].Figure || ps[1].Text != "Найдите $x^2$ на рисунке." || ps[0].Figure {
		t.Errorf("a problem with a picture: %+v", ps[1])
	}
	if ps[2].Number != 0 || ps[2].Err != nil || !strings.HasPrefix(ps[2].Text, "Задача без номера. а)") {
		t.Errorf("third %+v", ps[2])
	}
	if ps[3].Number != 4 || ps[3].Year != 2010 || ps[3].Grade != 10 || ps[0].Year != 0 {
		t.Errorf("a number with the year and the grade: %+v", ps[3])
	}
	if ps[4].Number != 7 || ps[4].Year != 0 || ps[4].Grade != 8 {
		t.Errorf("a number with the grade: %+v", ps[4])
	}
	for _, p := range ps {
		if strings.Contains(p.Text, "Иванов") || strings.Contains(p.Text, "Сложность") || strings.Contains(p.Text, "Тема") {
			t.Errorf("%d keeps the author, the difficulty or the topics: %q", p.ID, p.Text)
		}
	}
}

func prSourceHead(id, name string) string {
	return `<tr class="problemdetailssourcetablecell"><td class="problemdetailssourcetablecellhead" colspan=2>
<a class="componentboxlink" href="/view_by_source_new.php?parent=` + id + `">` + name + `</a>	</td></tr>`
}

func prSourceCell(k, v string) string {
	return `<tr class="problemdetailssourcetablecell">	<td class="problemdetailssourcetablecell">
` + k + `	</td>
	<td class="problemdetailssourcetablecell">
` + v + `	</td></tr>`
}

func TestParsePRSources(t *testing.T) {
	use := func(grade, extra string) string {
		return prSourceHead("204563", "") + prSourceHead("164898", "олимпиада") + prSourceCell("Название", "Московская математическая олимпиада") +
			prSourceHead("200267", "год") + prSourceCell("Номер", "67") + prSourceCell("Год", "2004") +
			prSourceHead("200289", "вариант") + prSourceCell("Класс", grade) + extra +
			prSourceHead("200295", "задача") + prSourceCell("Номер", "6")
	}
	page := `<h3>Источники и прецеденты использования</h3><table class="problemdetailssourcetable">` +
		use("11", "") + use("10", prSourceCell("День", "2")) +
		prSourceHead("204563", "") + prSourceHead("5", "олимпиада") + prSourceCell("Название", "Другая олимпиада") + prSourceHead("6", "задача") + prSourceCell("Номер", "3") +
		`</table>`
	got := ParsePRSources(page, 164898)
	want := []PRSource{{Year: 2004, Grade: 11, Number: 6}, {Year: 2004, Grade: 10, Number: 6, Other: true}}
	// The Sharygin olympiad has the head класс, a head задача with no link and a number such as 10.4.
	// The 2005 pages write the head as Задача, with a capital.
	shar := func(grade, head, number string) string {
		return prSourceHead("204563", "") + prSourceHead("206102", "олимпиада") + prSourceCell("Название", "Олимпиада по геометрии имени И.Ф. Шарыгина") +
			prSourceHead("216766", "год") + prSourceCell("Год", "2025") + prSourceHead("216770", "класс") + prSourceCell("Класс", grade) +
			`<tr class="problemdetailssourcetablecell"><td class="problemdetailssourcetablecellhead" colspan=2>
` + head + `	</td></tr>` + prSourceCell("Номер", number)
	}
	for _, c := range []struct {
		grade, head, number string
		want                PRSource
	}{
		{"10", "задача", "10.4", PRSource{Year: 2025, Grade: 10, Number: 4}},
		{"9", "задача", "10.4", PRSource{Year: 2025, Grade: 9, Other: true}},
		{"8", "задача", "3", PRSource{Year: 2025, Grade: 8, Number: 3}},
		{"9", "Задача", "4", PRSource{Year: 2025, Grade: 9, Number: 4}},
	} {
		if got := ParsePRSources(shar(c.grade, c.head, c.number), 206102); len(got) != 1 || got[0] != c.want {
			t.Errorf("grade %s, head %s, number %s: %+v, want %+v", c.grade, c.head, c.number, got, c.want)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("sources %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("source %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}
