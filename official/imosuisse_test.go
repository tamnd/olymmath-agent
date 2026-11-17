package official

import "testing"

func TestImosuissePapers(t *testing.T) {
	page := `<a href="/files/exams/final-round-2015-german.pdf">German</a> <a href="/files/exams/final-round-2015-french.pdf">French</a>
<a href = "/files/exams/selection-2026-english.pdf">English</a> <a href="/files/exams/second-round-2026-italian.pdf">Italian</a>
<a href="/files/exams/final-round-2015-german.pdf">German</a> <a href="/files/scripts/algebra.pdf">Script</a>`
	got := ImosuissePapers(page)
	want := []National{
		{Competition: "ch-smo", Year: 2015, URL: "https://imosuisse.github.io/files/exams/final-round-2015-german.pdf", Lang: Lang{"German", "de"}},
		{Competition: "ch-smo", Year: 2015, URL: "https://imosuisse.github.io/files/exams/final-round-2015-french.pdf", Lang: Lang{"French", "fr"}},
		{Competition: "ch-tst", Year: 2026, URL: "https://imosuisse.github.io/files/exams/selection-2026-english.pdf", Lang: Lang{"English", "en"}},
		{Competition: "ch-smo-r2", Year: 2026, URL: "https://imosuisse.github.io/files/exams/second-round-2026-italian.pdf", Lang: Lang{"Italian", "it"}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d papers, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
