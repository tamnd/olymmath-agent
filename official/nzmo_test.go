package official

import (
	"slices"
	"testing"
)

func TestNZMOPapers(t *testing.T) {
	page := `<li><a href="//www.mathsolympiad.org.nz/competitions/nzmo/files/problems.php?round=1&amp;year=2025">Round One problems</a>
<a href="//www.mathsolympiad.org.nz/competitions/nzmo/files/problems.php?round=1&amp;year=2025&amp;type=solutions">solutions</a></li>
<li><a href="https://www.mathsolympiad.org.nz/competitions/nzmo/files/problems.php?round=2&year=2019">Round Two problems</a></li>
<li><a href="//www.mathsolympiad.org.nz/competitions/nzmo/files/problems.php?round=1&amp;year=2025">again</a></li>`
	got := NZMOPapers(page)
	want := []National{
		{Competition: "nz-nzmo", Year: 2025, Paper: "round1", URL: "https://www.mathsolympiad.org.nz/competitions/nzmo/files/problems.php?round=1&year=2025"},
		{Competition: "nz-nzmo", Year: 2019, Paper: "round2", URL: "https://www.mathsolympiad.org.nz/competitions/nzmo/files/problems.php?round=2&year=2019"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("NZMOPapers = %+v", got)
	}
}

func TestNZMOCover(t *testing.T) {
	r := Readings["nzmoc"]
	// From 2020 the page of the instructions ends with its heading.
	got := r.cover([]string{"Instructions\n1. You have 3 hours.\n2. There are 5 problems.\nNZMO Round Two 2023 — Instructions\n", "NZMO Round Two 2023 — Problems\n1. Find all primes.\n"})
	if len(got) != 2 || got[0] != "" || got[1] != "NZMO Round Two 2023 — Problems\n1. Find all primes.\n" {
		t.Errorf("heading: got %q", got)
	}
	// The instructions of Round Two 2019 end with the rule "Do not turn over until told to do so."
	got = r.cover([]string{"2019 NZMO Round 2\nInstructions\n1. You have 3 hours to work on the exam.\n10. Do not turn over until told to do so.\nPage 1 of 2\n", "Problems\n1. A positive integer is called sparkly.\n"})
	if len(got) != 2 || got[0] != "" || got[1] != "Problems\n1. A positive integer is called sparkly.\n" {
		t.Errorf("2019: got %q", got)
	}
	// Round One 2019 is one page with no instructions.
	got = r.cover([]string{"NZMO Round One 2019\n1. How many positive integers less than 2019 are divisible by 18 or 21?\n"})
	if got[0] != "NZMO Round One 2019\n1. How many positive integers less than 2019 are divisible by 18 or 21?\n" {
		t.Errorf("no cover: got %q", got)
	}
}
