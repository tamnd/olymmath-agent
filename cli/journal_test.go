package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/review"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/store"
)

func TestWriteJournal(t *testing.T) {
	root := copyContent(t)
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the test content has %d errors", f.Errors())
	}
	c := canon.Candidate{Source: "evanchen", Origin: "H2", Zone: "Z3", Lang: "en", Name: "A journal, problem 5171", URL: "https://example.org/issue.pdf"}
	news := []journalLink{{at: "imo/2031/p5171", cand: c, page: 40}}
	if err := writeJournal(root, corpus, news); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "problems", "imo", "2031", "p5171.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"https://example.org/issue.pdf", "A journal, problem 5171"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the record has no %q:\n%s", want, b)
		}
	}
	corpus, f = content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the content has %d errors after om journal: %v", f.Errors(), f)
	}
	p := official.Owners(corpus)["imo/2031/p5171"]
	if p == nil || strings.TrimSpace(p.Statement) != "" {
		t.Errorf("the link-only record is missing or has a text: %+v", p)
	}
	if code, _, _ := run("journal", "-content", root, "nojournal"); code != ExitUsage {
		t.Errorf("an unknown journal gave %d", code)
	}
}

func TestJournalMarks(t *testing.T) {
	root := copyContent(t)
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the test content has %d errors", f.Errors())
	}
	j := archive.Journal{Comp: "imo", Name: func(i archive.Issue, m archive.Mark) string {
		return fmt.Sprintf("A journal Vol. %d, No. %d (%s %d), problem %s", i.Volume, i.Number, i.Month, i.Year, m.Label)
	}}
	dir := t.TempDir()
	write := func(name, body string) string {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return file
	}
	pdf := "https://example.org/v1n2.pdf"
	// Problem 14 is not there, so the run has a gap. The numbers of an issue need not be in order.
	good := write("good.tsv", "# year, volume, number, month, PDF, problem, page\n"+
		"2031\t1\t1\tJan\thttps://example.org/v1n1.pdf\t11\t3\n"+
		"2031\t1\t1\tJan\thttps://example.org/v1n1.pdf\t12\t3\n"+
		"2031\t1\t2\tFeb\t"+pdf+"\t15\t5\n"+
		"2031\t1\t2\tFeb\t"+pdf+"\t13\t4\n")
	var out, errs strings.Builder
	env := Env{Stdout: &out, Stderr: &errs}
	if code := journalMarks(env, root, corpus, "evanchen", j, good, "vision/claude@2.1.323 (claude-opus-5-5)", false); code != ExitOK {
		t.Fatalf("om journal -marks gave %d: %s", code, errs.String())
	}
	if !strings.Contains(errs.String(), "vol. 1 no. 2: no number from 14 to 14") || !strings.Contains(out.String(), "4 numbers, 0 already in the content, 1 gaps, 4 new problems") {
		t.Errorf("the run told:\n%s%s", out.String(), errs.String())
	}
	b, err := os.ReadFile(filepath.Join(root, "problems", "imo", "2031", "p15.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{pdf, "page: 5", "A journal Vol. 1, No. 2 (Feb 2031), problem 15", "extraction: vision/claude@2.1.323 (claude-opus-5-5)", "zone: Z3", "statement: sourced"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the record has no %q:\n%s", want, b)
		}
	}
	corpus, f = content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the content has %d errors after om journal -marks: %v", f.Errors(), f)
	}
	if p := official.Owners(corpus)["imo/2031/p15"]; p == nil || strings.TrimSpace(p.Statement) != "" {
		t.Errorf("the link-only record is missing or has a text: %+v", p)
	}
	// A second run finds the records and adds nothing.
	out.Reset()
	if code := journalMarks(env, root, corpus, "evanchen", j, good, "vision/claude@2.1.323 (claude-opus-5-5)", false); code != ExitOK || !strings.Contains(out.String(), "4 already in the content") {
		t.Errorf("the second run gave %d: %s", code, out.String())
	}
	// A number on two pages, or a number below the numbers of an earlier issue, stops the run.
	for _, body := range []string{
		"2032\t2\t1\tJan\t" + pdf + "\t21\t3\n2032\t2\t1\tJan\t" + pdf + "\t21\t4\n",
		"2032\t2\t1\tJan\t" + pdf + "\t21\t3\n2032\t2\t2\tFeb\t" + pdf + "\t20\t4\n",
	} {
		errs.Reset()
		if code := journalMarks(env, root, corpus, "evanchen", j, write("dup.tsv", body), "vision/x@1", false); code != ExitFindings || !strings.Contains(errs.String(), "duplicate") {
			t.Errorf("a duplicate gave %d: %s", code, errs.String())
		}
	}
	for _, body := range []string{"2032\t2\t1\tJan\t" + pdf + "\t21\n", "2032\t2\t1\tJan\t" + pdf + "\t21\t0\n", "2032\t2\t1\tJan\tftp://x\t21\t3\n", "2032\t2\t1\tJan\t" + pdf + "\t2a\t3\n"} {
		if code := journalMarks(env, root, corpus, "evanchen", j, write("bad.tsv", body), "vision/x@1", false); code != ExitUsage {
			t.Errorf("the bad line %q gave %d", body, code)
		}
	}
	if code, _, _ := run("journal", "-content", root, "-marks", good, "crux"); code != ExitUsage {
		t.Errorf("-marks with no -extraction gave %d", code)
	}
}

func TestLinks(t *testing.T) {
	root := copyContent(t)
	dir := t.TempDir()
	rows := []seed.Row{
		{Source: "evanchen", File: "f", Row: 0, ID: "501", Labels: []string{"imo"}, Year: 2032, Number: "p81", Lang: "ru", Statement: "A text that stays out.", Ref: "https://example.org/problem?id=501"},
		{Source: "evanchen", File: "f", Row: 1, ID: "502", Labels: []string{"imo"}, Year: 2032, Number: "p81", Lang: "ru", Statement: "The same position again.", Ref: "https://example.org/problem?id=502"},
		{Source: "evanchen", File: "f", Row: 2, ID: "503", Labels: []string{"no such olympiad"}, Year: 2032, Number: "p1", Ref: "https://example.org/problem?id=503"},
		{Source: "evanchen", File: "f", Row: 3, ID: "504", Labels: []string{"imo"}, Year: 2032, Number: "", Ref: "https://example.org/problem?id=504"},
		// A row that names its edition keeps its exact position. The edition must start with the year of the row.
		{Source: "evanchen", File: "f", Row: 4, ID: "505", Labels: []string{"imo"}, Year: 2032, Edition: "2032-fall", Number: "ja3", Ref: "https://example.org/problem?id=505"},
		{Source: "evanchen", File: "f", Row: 5, ID: "506", Labels: []string{"imo"}, Year: 2032, Edition: "2031-fall", Number: "ja4", Ref: "https://example.org/problem?id=506"},
		{Source: "evanchen", File: "f", Row: 6, ID: "507", Labels: []string{"imo"}, Year: 2032, Edition: "2032-fall", Number: "ja5/p1", Ref: "https://example.org/problem?id=507"},
		// The same page in another paper is one more appearance of the same problem.
		{Source: "evanchen", File: "f", Row: 7, ID: "505", Labels: []string{"imo"}, Year: 2032, Edition: "2032-fall", Number: "sa1", Ref: "https://example.org/problem?id=505"},
	}
	err := seed.WriteFile(dir, "evanchen", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := run("links", "-content", root, "-seed", dir, "imo-official"); code != ExitUsage {
		t.Errorf("a source of zone Z2 gave %d", code)
	}
	code, out, errs := run("links", "-content", root, "-seed", dir, "evanchen")
	if code != ExitOK || !strings.Contains(out, "2 new link-only problems") {
		t.Fatalf("om links gave %d %q %q", code, out, errs)
	}
	b, err := os.ReadFile(filepath.Join(root, "problems", "imo", "2032", "p81.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "https://example.org/problem?id=501") || !strings.Contains(string(b), "problem 501") || strings.Contains(string(b), "stays out") {
		t.Errorf("the record is not a link to problem 501:\n%s", b)
	}
	if corpus, f := content.Load(root); f.Errors() > 0 || official.Owners(corpus)["imo/2032/p81"] == nil || official.Owners(corpus)["imo/2032-fall/ja3"] == nil {
		t.Errorf("the content has %d errors, or no imo/2032/p81 or imo/2032-fall/ja3: %v", f.Errors(), f)
	}
	if b, err := os.ReadFile(filepath.Join(root, "editions", "imo", "2032-fall.yaml")); err != nil || !strings.HasPrefix(string(b), "year: 2032\n") {
		t.Errorf("the edition 2032-fall: %q, %v", b, err)
	}
	corpus, _ := content.Load(root)
	if p := official.Owners(corpus)["imo/2032-fall/sa1"]; p == nil || p.Locator() != "imo/2032-fall/ja3" {
		t.Errorf("the page of two papers is not one problem: %+v", p)
	}
	// A second run adds nothing: the positions are taken, and a page that the content links to is not linked again at another position.
	rows[7].Number = "sa2"
	err = seed.WriteFile(dir, "evanchen", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run("links", "-content", root, "-seed", dir, "evanchen"); code != ExitOK || !strings.Contains(out, "0 new link-only problems") || !strings.Contains(out, "problem in the content at another position 1") {
		t.Errorf("a second run gave %d %q %q", code, out, errs)
	}
	// With -appear, the page that the content links at another position gives one more appearance of that problem, a merge row and a review, and no second link.
	if code, out, errs := run("links", "-content", root, "-seed", dir, "-appear", "-date", "2032-01-02", "evanchen"); code != ExitOK || !strings.Contains(out, "0 new link-only problems, 1 appearances added") {
		t.Fatalf("om links -appear gave %d %q %q", code, out, errs)
	}
	corpus, f := content.Load(root)
	p := official.Owners(corpus)["imo/2032-fall/sa2"]
	if f.Errors() > 0 || p == nil || p.Locator() != "imo/2032-fall/ja3" || len(p.Statements) != 1 {
		t.Fatalf("the appearance is not on the problem of the page: %+v %v", p, f)
	}
	ms, err := review.Merges(root)
	if err != nil || len(ms) == 0 || ms[len(ms)-1].At != "imo/2032-fall/sa2" || ms[len(ms)-1].Into != "imo/2032-fall/ja3" {
		t.Fatalf("merges %+v %v", ms, err)
	}
	revs, _ := review.Load(root)
	if r := revs[ms[len(ms)-1].Review]; r == nil || r.Kind != "merge" || r.Decision != "accept" || r.By != "agent" || !strings.HasPrefix(r.Note, "Agent decision, not a person's review.") || !strings.Contains(r.Note, "problem 505") {
		t.Errorf("review %+v", r)
	}
	if code, out, _ := run("links", "-content", root, "-seed", dir, "-appear", "evanchen"); code != ExitOK || !strings.Contains(out, "0 appearances added") {
		t.Errorf("a second run of -appear gave %d %q", code, out)
	}
}

func TestJournalScan(t *testing.T) {
	pages := map[string]string{
		"/issue/1":    `<a class='crux-download-link' href='https://example.org/v1n1.pdf'>Download</a>`,
		"/v1n1.pdf":   "%PDF-1.4 a scan",
		"/issue/2":    `<p>no link</p>`,
		"/robots.txt": "User-agent: *\nAllow: /\n",
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := pages[r.URL.Path]
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
	j := archive.Journals["crux"]
	is := archive.Issue{Year: 1975, Volume: 1, Number: 1, Month: "Mar", Page: "https://example.org/issue/1"}
	row, err := journalScan(context.Background(), fc, "crux", j, &is)
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Split(strings.TrimSuffix(row, "\n"), "\t")
	if len(f) != 6 || strings.Join(f[:5], " ") != "1975 1 1 Mar https://example.org/v1n1.pdf" || len(f[5]) != 64 {
		t.Fatalf("row %q", row)
	}
	rc, err := st.Get(context.Background(), f[5])
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(b) != "%PDF-1.4 a scan" {
		t.Errorf("the store has %q", b)
	}
	is = archive.Issue{Year: 1975, Volume: 1, Number: 2, Month: "Apr", Page: "https://example.org/issue/2"}
	if _, err := journalScan(context.Background(), fc, "crux", j, &is); err == nil {
		t.Error("an issue page with no PDF link gave no error")
	}

	// An issue that is not a scan goes to the vision lane when its text layer is not readable.
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("no pdftotext")
	}
	pages["/issue/3"] = `<a class='crux-download-link' href='https://example.org/v31n1.pdf'>Download</a>`
	pages["/v31n1.pdf"] = string(onePagePDF(`(\351\351\351\351\351\351\351\351) Tj`))
	is = archive.Issue{Year: 2005, Volume: 31, Number: 1, Month: "Feb", Page: "https://example.org/issue/3"}
	row, err = journalScan(context.Background(), fc, "crux", j, &is)
	if err != nil || !strings.HasPrefix(row, "2005\t31\t1\tFeb\thttps://example.org/v31n1.pdf\t") {
		t.Errorf("an unreadable text layer gave %q, %v", row, err)
	}
}

// onePagePDF is a PDF of one page that shows text with a standard font in the WinAnsi encoding. pdftotext rebuilds the missing cross-reference table.
func onePagePDF(text string) []byte {
	stream := "BT /F1 12 Tf 72 720 Td " + text + " ET"
	return []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>\nendobj\n" +
		fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(stream), stream) +
		"5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n" +
		"trailer\n<< /Root 1 0 R >>\n%%EOF\n")
}
