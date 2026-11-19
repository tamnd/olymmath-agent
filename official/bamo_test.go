package official

import (
	"reflect"
	"testing"
)

func TestBAMOPapers(t *testing.T) {
	page := `<a href="/archives/examfiles/bamo2026-prob-sol260225.pdf">2026</a>
<a href="/archives/examfiles/BAMO2016ProblemsAndSolutions.pdf">2016</a>
<a href="/archives/examfiles/bamo2002problems-and-solns.pdf">2002</a>
<a href="/archives/examfiles/bamo99draft.pdf">1999</a>
<a href="/archives/examfiles/bamo2016-flyer.png">flyer</a>
<a href="/archives/examfiles/bamo99draft.pdf">1999</a>`
	want := []National{
		{Competition: "bamo", Year: 2026, URL: "https://www.bamo.org/archives/examfiles/bamo2026-prob-sol260225.pdf"},
		{Competition: "bamo", Year: 2016, URL: "https://www.bamo.org/archives/examfiles/BAMO2016ProblemsAndSolutions.pdf"},
		{Competition: "bamo", Year: 2002, URL: "https://www.bamo.org/archives/examfiles/bamo2002problems-and-solns.pdf"},
		{Competition: "bamo", Year: 1999, URL: "https://www.bamo.org/archives/examfiles/bamo99draft.pdf"},
	}
	if got := BAMOPapers(page); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}
