package keymatch

import (
	"fmt"
	"slices"
	"testing"
)

// The tests use made-up texts in the style of a problems.ru page and of its translation.
const (
	ruKey = "В треугольнике ABC точка $M$ – середина стороны $AC$, точка $P$ лежит на стороне $BC$. Отрезок $AP$ пересекает $BM$ в точке $O$. Оказалось, что BO = BP. Найдите отношение $OM$ : $PC$."
	enRow = "5. In triangle $ABC$, point $M$ is the midpoint of side $AC$, and point $P$ lies on side $BC$. Segment $AP$ intersects $BM$ at point $O$. If $BO = BP$, find the ratio $OM: PC$."
	ruNum = "На доске написаны числа от 1 до 2013. За один ход можно стереть три числа и написать их сумму. Какое число, меньшее 7, может остаться на доске?"
	enNum = "The numbers from 1 to 2013 are written on a board. In one move one may erase three numbers and write their sum. Which number less than seven can remain on the board?"
)

func tokens(f Features) []string {
	var out []string
	for t := range f.Tokens {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

func TestExtract(t *testing.T) {
	// A text and its translation give the same tokens: the numbers, the letters and the names of points.
	ru, en := Extract(ruKey), Extract(enRow)
	if got, want := tokens(en), tokens(ru); !slices.Equal(got, want) {
		t.Errorf("tokens of the translation = %v, want %v", got, want)
	}
	// The mark of the problem, "5.", is not a number of the text.
	if en.Tokens["n:5"] {
		t.Errorf("the mark of the problem gave a token")
	}
	// A Russian number word gives its number, as an English one does.
	if got, want := tokens(Extract(enNum)), tokens(Extract(ruNum)); !slices.Equal(got, want) {
		t.Errorf("number words: %v, want %v", got, want)
	}
	f := Extract("Prove that $x^2 \\le 2\\sqrt{y}$ and $\\sin x = 0.5$.")
	for _, w := range []string{"v:x", "v:y", "n:2", "n:0.5", "s:sqrt", "s:sin"} {
		if !f.Tokens[w] {
			t.Errorf("no token %s in %v", w, tokens(f))
		}
	}
	if !f.Grams["x ^ 2"] || !f.Grams["2 sqrt y"] {
		t.Errorf("grams %v", f.Grams)
	}
	// A Unicode square is a power, as in TeX.
	if !Extract("Решите уравнение x² = 4.").Grams["x ^ 2"] {
		t.Errorf("x² gave no power")
	}
	if !Extract("Докажите это.").Empty() {
		t.Errorf("a text with no formula has features")
	}
}

func key(at string, year int, text string) Key {
	return Key{At: at, Comp: "c", Year: year, F: Extract(text)}
}

func row(ref, comp string, year int, text string) Row {
	return Row{Ref: ref, Comp: comp, Year: year, F: Extract(text), Text: text}
}

const (
	other1 = "Найдите все натуральные $n$, для которых $n^3 + 7n$ делится на 19 и на 23."
	other2 = "Окружности $S_1$ и $S_2$ пересекаются в точках $K$ и $L$. Докажите, что $KL \\perp O_1 O_2$, где 1997 – номер года."
)

// fillers gives keys of other problems, so that the weights of the tokens are as in a competition with many problems.
func fillers(n int) []Key {
	var out []Key
	for i := range n {
		out = append(out, key(fmt.Sprintf("c/2012/p%d", i+1), 2012, fmt.Sprintf("Найдите $y$, если $y - %d = %d z$.", 30+i, 60+i)))
	}
	return out
}

func TestPairs(t *testing.T) {
	keys := append([]Key{key("c/2010/p1", 2010, ruKey), key("c/2010/p2", 2010, ruNum), key("c/2011/p1", 2011, other1), key("c/2011/p2", 2011, other2)}, fillers(8)...)
	m := NewMatcher(keys)
	m.Add(row("a#1", "c", 2010, enRow))
	m.Add(row("a#2", "", 0, enNum))
	// A row of another year, and a row of another competition, are not compared.
	m.Add(row("a#3", "c", 2009, enRow))
	m.Add(row("a#4", "d", 2010, enRow))
	ps := m.Pairs()
	got := map[string]string{}
	for _, p := range ps {
		got[p.At] = p.Ref + " " + p.Class
	}
	if got["c/2010/p1"] != "a#1 strong" || len(got) != 1 {
		t.Errorf("pairs %v", got)
	}
	// The open row shares 4 tokens with its key (1, 2013, 3 and 7), fewer than MinTokensOpen, so it is not placed.
	if _, ok := got["c/2010/p2"]; ok {
		t.Errorf("the open row with few tokens was placed")
	}
	if p := got["c/2011/p1"]; p != "" {
		t.Errorf("a key with no row got %s", p)
	}
	// The same row with its competition and year is placed.
	m = NewMatcher(keys)
	m.Add(row("a#2", "c", 2010, enNum))
	if ps := m.Pairs(); len(ps) != 1 || ps[0].At != "c/2010/p2" {
		t.Errorf("pairs of the labelled row %+v", ps)
	}
}

func TestPairsUnique(t *testing.T) {
	keys := []Key{key("c/2010/p1", 2010, ruKey), key("c/2011/p1", 2011, other1), key("c/2011/p2", 2011, other2)}
	// Two rows of one text do not compete for the key.
	m := NewMatcher(keys)
	m.Add(row("a#1", "c", 2010, enRow))
	m.Add(row("b#1", "c", 2010, enRow+" "))
	ps := m.Pairs()
	if len(ps) != 1 || ps[0].Ref != "a#1" {
		t.Fatalf("pairs %+v", ps)
	}
	// Two rows of two texts that come as near the key are not placed: neither is clearly the one.
	m = NewMatcher(keys)
	m.Add(row("a#1", "c", 2010, enRow))
	m.Add(row("b#1", "c", 2010, "Another text: in triangle $ABC$ the point $M$ is on $AC$, $P$ is on $BC$, $AP$ meets $BM$ at $O$, $BO = BP$; find $OM : PC$ again, said the teacher to the class."))
	if ps := m.Pairs(); len(ps) != 0 {
		t.Errorf("pairs with a close second row %+v", ps)
	}
}

func TestFits(t *testing.T) {
	k := Key{Comp: "tot", Year: 2005, Season: true}
	for _, c := range []struct {
		r    Row
		want bool
	}{
		{Row{Comp: "tot", Year: 2005}, true},
		{Row{Comp: "tot", Year: 2004}, true},
		{Row{Comp: "tot", Year: 2003}, false},
		{Row{Comp: "", Year: 0}, true},
		{Row{Comp: "imo", Year: 2005}, false},
	} {
		if got := fits(c.r, k); got != c.want {
			t.Errorf("fits(%+v) = %v", c.r, got)
		}
	}
	if fits(Row{Comp: "c", Year: 2004}, Key{Comp: "c", Year: 2005}) {
		t.Errorf("a row of the year before fits a key with no seasons")
	}
}
