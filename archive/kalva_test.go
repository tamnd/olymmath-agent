package archive

import (
	"strings"
	"testing"
)

const kalvaPage = `<h3>37th Vietnam 1999 problems</h3>
<TABLE><TR><TD>
<b>A1.</b>&nbsp;  Find all real solutions to (1 + 4<sup>2x-y</sup>)(5<sup>y-2x+1</sup>) = 2<sup>2x-y+1</sup> + 1.
		</TD>
<!--
<td><a href="vsoln/vsol991.html"><IMG SRC="../sol1.gif"></a></td>
-->
	</TR><TR><td>
<b>A2.</b>&nbsp; The sequence is a<sub>n+2</sub> = 3a<sub>n+1</sub> - a<sub>n</sub> for n &ge; 1.
		</td></TR><TR><td>
<b>B1.</b>&nbsp; Show the figure <IMG SRC="v1.gif">.
		</td></TR></TABLE>`

func TestParseKalva(t *testing.T) {
	ps := ParseKalva(kalvaPage)
	if len(ps) != 3 {
		t.Fatalf("%d problems", len(ps))
	}
	for i, n := range []string{"1", "2", "3"} {
		if ps[i].Number != n {
			t.Errorf("problem %d has number %q", i, ps[i].Number)
		}
	}
	if want := "Find all real solutions to (1 + $4^{2x-y}$)($5^{y-2x+1}$) = $2^{2x-y+1}$ + 1."; ps[0].Text != want {
		t.Errorf("A1:\n got %q\nwant %q", ps[0].Text, want)
	}
	if want := "The sequence is $a_{n+2}$ = $3a_{n+1}$ - $a_n$ for n ≥ 1."; ps[1].Text != want {
		t.Errorf("A2:\n got %q\nwant %q", ps[1].Text, want)
	}
	if ps[2].Err == nil {
		t.Error("B1 has a picture and no error")
	}
}

func TestKalva(t *testing.T) {
	docs := Kalva(1972, 1973)
	var urls []string
	for _, d := range docs {
		urls = append(urls, d.Label+" "+d.URL)
	}
	got := strings.Join(urls, "\n")
	want := "International Mathematical Olympiad https://prase.cz/kalva/imo/imo72.html\nInternational Mathematical Olympiad https://prase.cz/kalva/imo/imo73.html\nUSAMO https://prase.cz/kalva/usa/usa72.html\nUSAMO https://prase.cz/kalva/usa/usa73.html\nVietnam National Olympiad https://prase.cz/kalva/vietnam/viet72.html"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
