package cli

import (
	"archive/zip"
	"bytes"
	"slices"
	"testing"

	"github.com/tamnd/olymmath-agent/official"
)

func TestUniboYear(t *testing.T) {
	for name, want := range map[string]int{
		"Archimede2022.zip":        2022,
		"Cesenatico1997.zip":       1997,
		"ces16-sol.pdf":            2016,
		"SolFeb25.pdf":             2025,
		"GaraPrime1617.pdf":        2017,
		"Febbraio-2014.zip":        2014,
		"soluzioni2022-1.pdf":      2022,
		"Soluzioni_gara_prime.pdf": 0,
	} {
		if got := uniboYear(name); got != want {
			t.Errorf("uniboYear(%q) = %d, want %d", name, got, want)
		}
	}
}

func TestUniboMembers(t *testing.T) {
	for _, c := range []struct {
		code  string
		names []string
		want  []hrPaper
	}{
		{"it-archimede", []string{"Archimede1999/Soluzionibiennio1999.pdf", "Archimede1999/Soluzionitriennio1999.pdf", "Archimede1999/TestoBiennio1999.pdf", "Archimede1999/TestoTriennio1999.pdf"},
			[]hrPaper{{kind: "it-archimede", variant: "biennio", member: "Archimede1999/TestoBiennio1999.pdf"}, {kind: "it-archimede", variant: "triennio", member: "Archimede1999/TestoTriennio1999.pdf"}}},
		// The letter to the schools with both papers goes when the zip file has the papers apart, and so do the answer grids.
		{"it-archimede", []string{"lettera.pdf", "archimede_biennio_2018.pdf", "archimede_biennio_2018_risp.pdf", "GiochiArchimede2018.pdf", "archimede_triennio_2018.pdf"},
			[]hrPaper{{kind: "it-archimede", variant: "biennio", member: "archimede_biennio_2018.pdf"}, {kind: "it-archimede", variant: "triennio", member: "archimede_triennio_2018.pdf"}}},
		{"it-archimede", []string{"GiochiArchimede2011.pdf", "soluzioni2011.pdf"}, []hrPaper{{kind: "it-archimede", member: "GiochiArchimede2011.pdf"}}},
		{"it-cesenatico", []string{"GaraIndividuale.pdf", "GaraIndividualeEnglish.pdf", "GaraPubblico.pdf", "SoluzioniGaraSquadre.pdf"}, []hrPaper{{kind: "it-cesenatico", member: "GaraIndividuale.pdf"}}},
		{"it-cesenatico", []string{"ces05.pdf", "febb05.pdf"}, []hrPaper{{kind: "it-cesenatico", member: "ces05.pdf"}}},
		{"it-febbraio", []string{"Febbraio97-Soluzioni.pdf", "Febbraio97-Testo.pdf"}, []hrPaper{{kind: "it-febbraio", member: "Febbraio97-Testo.pdf"}}},
		{"it-febbraio", []string{"soluzionifebb11.pdf"}, []hrPaper{{kind: "it-febbraio", member: "soluzionifebb11.pdf"}}},
		{"it-febbraio", []string{"readme.txt"}, nil},
	} {
		if got := uniboMembers(c.code, c.names); !slices.Equal(got, c.want) {
			t.Errorf("uniboMembers(%s, %q) = %+v, want %+v", c.code, c.names, got, c.want)
		}
	}
}

func TestUniboPlace(t *testing.T) {
	prob := func(n int, section string) official.HrProblem {
		return official.HrProblem{N: n, Section: section}
	}
	ats, names := uniboPlace(hrPaper{kind: "it-archimede", year: 2009}, []official.HrProblem{prob(1, ""), prob(1, "B iennio"), prob(2, "Nome Cognome"), prob(1, "t riennio")})
	if want := []string{"", "it-archimede/2009/b1", "it-archimede/2009/b2", "it-archimede/2009/t1"}; !slices.Equal(ats, want) {
		t.Errorf("uniboPlace archimede = %q, want %q", ats, want)
	}
	if names[1] != "Olimpiadi della Matematica, Giochi di Archimede 2009, gara biennio" || names[3] != "Olimpiadi della Matematica, Giochi di Archimede 2009, gara triennio" {
		t.Errorf("uniboPlace archimede names = %q", names)
	}
	ats, _ = uniboPlace(hrPaper{kind: "it-archimede", variant: "triennio", year: 2013}, []official.HrProblem{prob(1, "Nome Cognome Classe"), prob(2, "Nome Cognome Classe")})
	if want := []string{"it-archimede/2013/t1", "it-archimede/2013/t2"}; !slices.Equal(ats, want) {
		t.Errorf("uniboPlace archimede triennio = %q, want %q", ats, want)
	}
	// The numbered instructions before the first part of a paper of February have no place.
	ats, names = uniboPlace(hrPaper{kind: "it-febbraio", year: 2015}, []official.HrProblem{prob(1, ""), prob(1, "Problemi a risposta multipla"), prob(13, "Problemi a risposta numerica")})
	if want := []string{"", "it-febbraio/2015/p1", "it-febbraio/2015/p13"}; !slices.Equal(ats, want) {
		t.Errorf("uniboPlace febbraio = %q, want %q", ats, want)
	}
	if names[1] != "Olimpiadi della Matematica, Gara di febbraio 2015" {
		t.Errorf("uniboPlace febbraio name = %q", names[1])
	}
	ats, _ = uniboPlace(hrPaper{kind: "it-cesenatico", year: 2015}, []official.HrProblem{prob(1, ""), prob(2, "")})
	if want := []string{"it-cesenatico/2015/p1", "it-cesenatico/2015/p2"}; !slices.Equal(ats, want) {
		t.Errorf("uniboPlace cesenatico = %q, want %q", ats, want)
	}
}

func TestUniboRules(t *testing.T) {
	if r := uniboRules(hrPaper{kind: "it-classi-prime", year: 2020}); !r.Columns {
		t.Error("the classi prime of 2020 are in two columns")
	}
	if r := uniboRules(hrPaper{kind: "it-classi-prime", year: 2024}); r.Columns {
		t.Error("the classi prime of 2024 are in one column")
	}
	if r := uniboRules(hrPaper{kind: "it-febbraio", year: 2024}); r.First || r.Section == nil {
		t.Error("the papers of February have parts that start at any number")
	}
	r := uniboRules(hrPaper{kind: "it-archimede", year: 2013})
	if !r.Columns || !r.Roman || r.Accents == nil {
		t.Error("the Giochi di Archimede are in two columns of Computer Modern")
	}
	for _, s := range []string{"1) La prova consiste di 20 problemi", "3) Per ciascuno dei problemi devi trascrivere"} {
		if !r.Not.MatchString(s) {
			t.Errorf("uniboNot does not match %q", s)
		}
	}
	for s, want := range map[string]string{"I Giochi di Archimede - Gara Triennio": "Triennio", "I G iochi di A rchim ede - G ara B iennio": "B iennio"} {
		if m := r.Section.FindStringSubmatch(s); m == nil || m[1] != want {
			t.Errorf("uniboArchimedeSection(%q) = %q, want %q", s, m, want)
		}
	}
}

func TestUniboFix(t *testing.T) {
	for in, want := range map[string]string{
		"$F$ . Siano $a = 2255$": "Siano $a = 2255$",
		".$F$ Nel ristorante":    "Nel ristorante",
		"$F$. In un triangolo":   "In un triangolo",
		"Per dipingere un muro":  "Per dipingere un muro",
		"$$F$$\n\n. Siano dati":  "Siano dati",
	} {
		p := official.HrProblem{}
		p.Text = in
		if uniboFix(&p); p.Text != want {
			t.Errorf("uniboFix(%q) = %q, want %q", in, p.Text, want)
		}
	}
}

func TestZipMember(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{"Archimede1999/": "", "Archimede1999/Testo.pdf": "testo", "__MACOSX/Archimede1999/._Testo.pdf": "x", ".DS_Store": "x", "sol.pdf": "sol"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	names, err := zipNames(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(names)
	if want := []string{"Archimede1999/Testo.pdf", "sol.pdf"}; !slices.Equal(names, want) {
		t.Errorf("zipNames = %q, want %q", names, want)
	}
	if b, err := zipMember(buf.Bytes(), "Archimede1999/Testo.pdf"); err != nil || string(b) != "testo" {
		t.Errorf("zipMember = %q, %v", b, err)
	}
	if _, err := zipMember(buf.Bytes(), "none.pdf"); err == nil {
		t.Error("zipMember of a missing file has no error")
	}
	if _, err := zipNames([]byte("no zip")); err == nil {
		t.Error("zipNames of no zip file has no error")
	}
}
