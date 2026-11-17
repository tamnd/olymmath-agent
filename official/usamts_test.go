package official

import "testing"

func TestUSAMTSPapers(t *testing.T) {
	page := `<a href="https://files.usamts.org/Problems_37_1.pdf">Problems</a>
<a href="https://files.usamts.org/Solutions_37_1.pdf">Solutions</a>
<a href="https://files.usamts.org/Rubric_37_1.pdf">Rubric</a>
<a href="https://files.usamts.org/Problems_22_2.pdf">Problems</a>
<a href="https://files.usamts.org/Problems_10_4.pdf">Problems</a>
<a href="https://files.usamts.org/Problems_10_4.pdf">Problems</a>`
	got := USAMTSPapers(page)
	want := []National{
		{Competition: "usamts", Year: 2025, Paper: "round1", URL: "https://files.usamts.org/Problems_37_1.pdf"},
		{Competition: "usamts", Year: 1998, Paper: "round4", URL: "https://files.usamts.org/Problems_10_4.pdf"},
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
