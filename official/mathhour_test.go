package official

import (
	"reflect"
	"testing"
)

func TestMathHourPapers(t *testing.T) {
	page := `<a href="./MHO2012_5-7_1-5.pdf">Grades 5-7</a>
<a href="./MHO2012_8-10_6-7.pdf">Grades 8-10, extra</a>
<a href="19main67.pdf">Grades 6-7</a>
<a href="13adv810.pdf">Grades 8-10, extra</a>
<a href="./2025-67-extra.pdf">Grades 6-7, extra</a>
<a href="22problems.pdf">Problems</a>
<a href="19main67.pdf">Grades 6-7</a>`
	a := "https://sites.math.washington.edu/~mathcircle/olympiad/archive/"
	want := []National{
		{Competition: "math-hour", Year: 2012, Paper: "grades6-7", URL: a + "olympiad_2012/MHO2012_5-7_1-5.pdf"},
		{Competition: "math-hour", Year: 2012, Paper: "grades8-10", URL: a + "olympiad_2012/MHO2012_8-10_6-7.pdf"},
		{Competition: "math-hour", Year: 2019, Paper: "grades6-7", URL: a + "olympiad_2019/19main67.pdf"},
		{Competition: "math-hour", Year: 2013, Paper: "grades8-10", URL: a + "olympiad_2013/13adv810.pdf"},
		{Competition: "math-hour", Year: 2025, Paper: "grades6-7", URL: a + "olympiad_2025/2025-67-extra.pdf"},
		{Competition: "math-hour", Year: 2022, URL: a + "olympiad_2022/22problems.pdf"},
	}
	if got := MathHourPapers(page); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if n := len(MathHourListings); n != 12 {
		t.Errorf("%d listings, want 12", n)
	}
}
