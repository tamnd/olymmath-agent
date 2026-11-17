package archive

import "testing"

func TestImosuisseDocs(t *testing.T) {
	index := `    {
        id: "2015-3-07",
        topic: "A",
    },
    {
        id: "1997-4-01",
    },
    {
        id: "2022-MEMO-T3",
    },
    {
        id: "2026-2-N1",
    },
    {
        id: "2015-3-07",
    },`
	got := ImosuisseDocs(index, 1998, 2030)
	want := []Doc{
		{Year: 2015, URL: imosuisseTeX + "2015-3-07.tex", Label: "ch-smo", Number: "7"},
		{Year: 2026, URL: imosuisseTeX + "2026-2-N1.tex", Label: "ch-smo-r2", Number: "Z1"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("doc %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if l, ok := Lanes["imosuisse"]; !ok || l.Index != ImosuisseIndex || l.List == nil || l.Docs != nil {
		t.Errorf("the lane of imosuisse is %+v", l)
	}
}

func TestParseImosuisse(t *testing.T) {
	tex := "Let $a,b,c$ be real numbers such that\n\\[\n    \\frac{a}{b+c} = 1.\n\\]\nDetermine all values.\n"
	ps := ParseImosuisse(tex)
	if len(ps) != 1 || ps[0].Err != nil || ps[0].Number != "" {
		t.Fatalf("got %+v", ps)
	}
	if want := "Let $a,b,c$ be real numbers such that \\[ \\frac{a}{b+c} = 1. \\] Determine all values."; ps[0].Text != want {
		t.Errorf("got %q, want %q", ps[0].Text, want)
	}
	if ps := ParseImosuisse("A figure \\includegraphics{a.png} and \\begin{tikzpicture}\\end{tikzpicture}."); len(ps) != 1 || ps[0].Err == nil || ps[0].Text != "" {
		t.Errorf("a figure: got %+v", ps)
	}
	if ps := ParseImosuisse("  \n"); len(ps) != 0 {
		t.Errorf("an empty file: got %+v", ps)
	}
}
