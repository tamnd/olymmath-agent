package cli

import (
	"reflect"
	"testing"

	"github.com/tamnd/olymmath-agent/official"
)

func TestOMSemPlace(t *testing.T) {
	ats, names := omSemPlace(hrPaper{kind: "pl-om-r2", year: 2024}, []official.HrProblem{{N: 4}, {N: 0}})
	if want := []string{"pl-om-r2/2024/p4", ""}; !reflect.DeepEqual(ats, want) {
		t.Errorf("omSemPlace places at %q, want %q", ats, want)
	}
	if names[0] != "Polish Mathematical Olympiad 2024, second round paper" {
		t.Errorf("omSemPlace names the paper %q", names[0])
	}
	if _, names := omSemPlace(hrPaper{kind: "pl-om-r1", year: 2003}, []official.HrProblem{{N: 1}}); names[0] != "Polish Mathematical Olympiad 2003, first round paper" {
		t.Errorf("omSemPlace names the paper %q", names[0])
	}
}

func TestOMSemPaperRules(t *testing.T) {
	r := omSemPaperRules(hrPaper{kind: "pl-om-r2", year: 2024})
	if !r.Columns || r.Lossy != nil {
		t.Errorf("2024 round 2: Columns %v, Lossy %v, want two columns and no lossy fonts", r.Columns, r.Lossy)
	}
	r = omSemPaperRules(hrPaper{kind: "pl-om-r1", year: 2010})
	if r.Columns || r.Lossy == nil || !r.Lossy.MatchString("PLRoman10") {
		t.Errorf("2010 round 1: Columns %v, Lossy %v, want one column and every font lossy", r.Columns, r.Lossy)
	}
	r = omSemPaperRules(hrPaper{kind: "pl-om-r1", year: 2024})
	if r.Columns || r.Lossy != nil {
		t.Errorf("2024 round 1: Columns %v, Lossy %v, want the plain rules", r.Columns, r.Lossy)
	}
}

func TestOMSemRules(t *testing.T) {
	for line, not := range map[string]bool{
		"1. Czas trwania zawodów: 5 godzin.":                       true,
		"3. W czasie zawodów nie wolno korzystać z kalkulatorów.":  true,
		"4. Każdy punkt okręgu pomalowano jednym z dwóch kolorów.": false,
		"2. Uczestnicy turnieju rozegrali po jednej partii.":       false,
	} {
		if got := omSemRules.Not.MatchString(line); got != not {
			t.Errorf("Not matches %q: %v, want %v", line, got, not)
		}
	}
	for _, line := range []string{"LXXV Olimpiada Matematyczna", "Zawody drugiego stopnia", "Informacje dla uczestnika zawodów", "12 stycznia 2024 r."} {
		if !omSemRules.Stop.MatchString(line) {
			t.Errorf("Stop does not match %q", line)
		}
	}
}

func TestOMSemFix(t *testing.T) {
	p := &official.HrProblem{}
	p.Text = "Dowieść, że $$<)APB+< )CPD = 180^\\circ.$$ Liczby $a<b$ są dodatnie, a $<)\\alpha$ jest ostry."
	omSemHarvest.fix(p)
	if want := "Dowieść, że $$\\angle APB+\\angle CPD = 180^\\circ.$$ Liczby $a<b$ są dodatnie, a $\\angle \\alpha$ jest ostry."; p.Text != want {
		t.Errorf("fix gives %q, want %q", p.Text, want)
	}
}
