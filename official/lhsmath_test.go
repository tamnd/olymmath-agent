package official

import (
	"reflect"
	"testing"
)

func TestLHSMathPapers(t *testing.T) {
	page := `<a href="Archive/2017/problems/Individual_Round_2017_prob.pdf">Problems</a>
<a href="Archive/2017/solutions/Individual_Round_2017_sol.pdf">Solutions</a>
<a href="Archive/2017/problems/Scramble_Round_prob.pdf">Problems</a>
<a href="Archive/2016/problems/2016 Guts Problems.pdf">Problems</a>
<a href="Archive/2018_spring/problems/individual.pdf">Problems</a>
<a href="Archive/2025_GLMT/GLMT 2025 Indiv Round.pdf">Problems</a>
<a href="Archive/2025_GLMT/GLMT 2025 Team Solutions.pdf">Solutions</a>
<a href="Archive/2017/problems/Individual_Round_2017_prob.pdf">Problems</a>`
	want := []National{
		{Competition: "lmt", Year: 2017, Paper: "individual", URL: "https://www.lhsmath.org/LMT/Archive/2017/problems/Individual_Round_2017_prob.pdf"},
		{Competition: "lmt", Year: 2017, Paper: "scramble", URL: "https://www.lhsmath.org/LMT/Archive/2017/problems/Scramble_Round_prob.pdf"},
		{Competition: "lmt", Year: 2016, Paper: "guts", URL: "https://www.lhsmath.org/LMT/Archive/2016/problems/2016%20Guts%20Problems.pdf"},
		{Competition: "glmt", Year: 2025, Paper: "individual", URL: "https://www.lhsmath.org/LMT/Archive/2025_GLMT/GLMT%202025%20Indiv%20Round.pdf"},
	}
	if got := LHSMathPapers(page); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	for i, w := range []string{"2017-spring", "2017-spring", "2016-spring", "2025"} {
		if got := LHSMathEdition(want[i]); got != w {
			t.Errorf("edition of %s: got %s, want %s", want[i].URL, got, w)
		}
	}
	if n := len(LHSMathListings); n != 9 {
		t.Errorf("%d listings, want 9", n)
	}
}
