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
	if _, ok := PRYear("Отборочный тур"); ok {
		t.Fatal("a tour is not a year")
	}
	for name, want := range map[string][3]int{"8 класс": {8, 0, 1}, "11 класс. Первый день": {11, 1, 1}, "11 класс. Второй день": {11, 2, 1}, "Устный тур": {0, 0, 0}} {
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
		prItem("504", "&nbsp;(#2010.10.4)", `<p/>Задача с полным номером.`)
	ps := ParsePRListing(page)
	if len(ps) != 4 {
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
	for _, p := range ps {
		if strings.Contains(p.Text, "Иванов") || strings.Contains(p.Text, "Сложность") || strings.Contains(p.Text, "Тема") {
			t.Errorf("%d keeps the author, the difficulty or the topics: %q", p.ID, p.Text)
		}
	}
}
