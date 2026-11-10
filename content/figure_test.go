package content

import (
	"slices"
	"testing"
)

func TestFigureRefs(t *testing.T) {
	text := "See ![Figure 1](p1.fig1.svg) and ![](/problems/x/p2.fig1.png) and ![a](https://x.org/a.png) and [a link](p1.md)."
	got := FigureRefs(text)
	want := []string{"p1.fig1.svg", "/problems/x/p2.fig1.png", "https://x.org/a.png"}
	if !slices.Equal(got, want) {
		t.Errorf("refs %q", got)
	}
	for ref, path := range map[string]string{
		"p1.fig1.svg":             "problems/imo/2024/p1.fig1.svg",
		"../2023/p1.fig1.svg":     "problems/imo/2023/p1.fig1.svg",
		"/problems/x/p2.fig1.png": "problems/x/p2.fig1.png",
		"https://x.org/a.png":     "",
	} {
		if got := FigurePath("problems/imo/2024/p1.md", ref); got != path {
			t.Errorf("FigurePath(%q) = %q, want %q", ref, got, path)
		}
	}
	got2 := MapFigures(text, func(ref string) string { return "<" + ref + ">" })
	if want := "See ![Figure 1](<p1.fig1.svg>) and ![](</problems/x/p2.fig1.png>) and ![a](<https://x.org/a.png>) and [a link](p1.md)."; got2 != want {
		t.Errorf("MapFigures: %s", got2)
	}
}

func TestIsFigureName(t *testing.T) {
	for name, ok := range map[string]bool{
		"p1.fig1.svg": true, "p1.fig12.png": true, "0J8TP3MFS9.fig2.svg": true,
		"p1.fig0.svg": false, "p1.fig1.jpg": false, "p1.svg": false, "a/p1.fig1.svg": false, "index.html": false,
	} {
		if IsFigureName(name) != ok {
			t.Errorf("IsFigureName(%q) = %v", name, !ok)
		}
	}
}

func TestProblemFigures(t *testing.T) {
	p := &Problem{Path: "problems/imo/2024/p1.md", Statement: "![1](p1.fig1.svg) ![2](p1.fig2.png) ![1 again](p1.fig1.svg) ![web](https://x.org/a.png)"}
	if got := p.Figures(); !slices.Equal(got, []string{"problems/imo/2024/p1.fig1.svg", "problems/imo/2024/p1.fig2.png"}) {
		t.Errorf("figures %q", got)
	}
}

func TestAddToStatement(t *testing.T) {
	front := "---\ntag: X\n---\n"
	for _, c := range []struct{ name, body, id, want string }{
		{"one statement", "The text.\n", "S1", "The text.\n\nFIG\n"},
		{"before the other statements", "The text.\n\n<!-- S2 -->\nLe texte.\n", "S1", "The text.\n\nFIG\n\n<!-- S2 -->\nLe texte.\n"},
		{"a later statement", "The text.\n\n<!-- S2 -->\nLe texte.\n", "S2", "The text.\n\n<!-- S2 -->\nLe texte.\n\nFIG\n"},
		{"an empty body", "", "S1", "FIG\n"},
	} {
		got, err := AddToStatement([]byte(front+c.body), c.id, "S1", "FIG")
		if err != nil || string(got) != front+c.want {
			t.Errorf("%s: %q %v", c.name, got, err)
		}
	}
	if _, err := AddToStatement([]byte(front+"The text.\n"), "S3", "S1", "FIG"); err == nil {
		t.Error("no error for a statement that is not in the body")
	}
}
