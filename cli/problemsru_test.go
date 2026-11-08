package cli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/store"
)

func TestPRPosition(t *testing.T) {
	for _, c := range []struct {
		g, d, n int
		want    string
	}{{8, 0, 1, "p81"}, {11, 1, 6, "p116"}, {11, 2, 3, "d113"}, {0, 0, 4, "p4"}} {
		got := prPosition(c.g, c.d, c.n)
		if got != c.want || seed.Position(got) != c.want {
			t.Errorf("prPosition(%d, %d, %d) = %q, want %q", c.g, c.d, c.n, got, c.want)
		}
	}
}

// prSite routes the requests of the fetch client to a handler, so that the URLs of problems.ru stay as they are.
type prSite struct{ h http.Handler }

func (s prSite) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	s.h.ServeHTTP(w, r)
	return w.Result(), nil
}

func prBox(links ...string) string {
	return `<a class="componentboxheaderlink" href="/view_by_source_new.php?parent=1">Олимпиада</a>` + strings.Join(links, "")
}

func prLink(id, name, count string) string {
	return `<a class="componentboxlink" href="/view_by_source_new.php?parent=` + id + `">` + name + `</a> (` + count + ` задач)`
}

func prListing(items ...[2]string) string {
	var b strings.Builder
	for _, it := range items {
		num := ""
		if it[1] != "" {
			num = `&nbsp;(#` + it[1] + `)`
		}
		b.WriteString(`<td class="problemsmallnumber"><a href="/view_problem_details_new.php?id=` + it[0] + `">` + it[0] + `</a>` + num + `</td>` +
			`<td class="problemsmalldifficulty">x</td></table><p/>Задача ` + it[0] + `.<table class="problemfootertable"></table>`)
	}
	b.WriteString(` [Всего задач: ` + string(rune('0'+len(items))) + `]`)
	return b.String()
}

// prSources is the table of sources of the page of a problem with one use in the olympiad of node 1. A non-empty key adds a row to the paper.
func prSources(year, grade, key, number string) string {
	head := func(id, name string) string {
		return `<td class="problemdetailssourcetablecellhead" colspan=2><a class="componentboxlink" href="/view_by_source_new.php?parent=` + id + `">` + name + `</a></td>`
	}
	cell := func(k, v string) string {
		return `<td class="problemdetailssourcetablecell">` + k + `</td><td class="problemdetailssourcetablecell">` + v + `</td>`
	}
	extra := ""
	if key != "" {
		extra = cell(key, "2")
	}
	return head("1", "олимпиада") + head("2", "год") + cell("Год", year) + head("3", "вариант") + cell("Класс", grade) + extra + head("4", "задача") + cell("Номер", number)
}

func TestSeedProblemsRu(t *testing.T) {
	pages := map[string]string{
		"/robots.txt": "User-agent: *\nDisallow: /show_document.php?\n",
		"parent=1":    prBox(prLink("10", "2020 год", "3"), prLink("20", "2019 год", "4"), prLink("30", "2018 год", "2"), prLink("40", "2017 год", "3")),
		"parent=10":   prBox(prLink("11", "11 класс. Первый день", "1"), prLink("12", "11 класс. Второй день", "1"), prLink("13", "Устный тур", "1")),
		"parent=11":   prListing([2]string{"101", "2020.11.1"}, [2]string{"103", "2019.11.2"}),
		"parent=12":   prListing([2]string{"102", "1"}),
		// 2019 has no papers and its numbers start again, so it gives no row.
		"parent=20": prBox() + prListing([2]string{"201", "1"}, [2]string{"202", "2"}, [2]string{"203", "1"}),
		"parent=30": prBox() + prListing([2]string{"301", "1"}, [2]string{"302", "2"}),
		// 2017 gives no numbers in its listing. The page of a problem gives it, unless it names more than the grade.
		"parent=40": prBox(prLink("41", "9 класс", "3")),
		"parent=41": prListing([2]string{"401", ""}, [2]string{"402", ""}, [2]string{"403", ""}),
		"id=401":    prSources("2017", "9", "", "2"),
		"id=402":    prSources("2017", "9", "День", "1"),
		"id=403":    prSources("2016", "9", "", "3"),
	}
	var hits int
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key = r.URL.RawQuery
		}
		p, ok := pages[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, p)
	})
	st, err := store.Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	fc := fetch.New(st)
	fc.Delay = 0
	fc.HTTP = &http.Client{Transport: prSite{h}}
	dir := t.TempDir()
	kept, skipped, err := seedProblemsRu(context.Background(), fc, dir, 1, 2017, 2020, time.Hour, true, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if kept != 5 || skipped != 7 {
		t.Errorf("kept %d, skipped %d", kept, skipped)
	}
	got := map[string]string{}
	err = seed.Read(dir, problemsRu, func(r seed.Row) error {
		got[r.ID] = r.Number
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"101": "p111", "102": "d111", "301": "p1", "302": "p2", "401": "p92"}
	if len(got) != len(want) {
		t.Errorf("rows %v", got)
	}
	for id, n := range want {
		if got[id] != n {
			t.Errorf("%s: %q, want %q", id, got[id], n)
		}
	}
	// A second run in the max age uses the raw store and sends no request.
	before := hits
	if _, _, err := seedProblemsRu(context.Background(), fc, dir, 1, 2017, 2020, time.Hour, true, io.Discard); err != nil || hits != before {
		t.Errorf("a second run sent %d requests: %v", hits-before, err)
	}
}
