package official

import (
	"reflect"
	"testing"
)

func TestPurpleCometListings(t *testing.T) {
	got := purpleCometListings(2019, 2020)
	want := []string{"https://purplecomet.org/contest/2019/MS", "https://purplecomet.org/contest/2019/HS", "https://purplecomet.org/contest/2020/MS", "https://purplecomet.org/contest/2020/HS"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPurpleCometPapers(t *testing.T) {
	// The heading and the frame of https://purplecomet.org/contest/2019/HS, read on 2026-11-16, and of the middle school page, with a second frame of the same paper.
	page := `<p><a href="https://purplecomet.org/answers">Back to Old Contests</a></p>
            <h3 class="text-4xl my-4">2019 HS Contest Questions</h3>
                            <iframe src="https://purplecomet.org/files/2019HS_English.pdf" class="w-11/12 h-screen"/>
            <h3 class="text-4xl my-4">2019 MS Contest Questions</h3>
                            <iframe src="https://purplecomet.org/files/2019MS_English.pdf" class="w-11/12 h-screen"/>
            <h3 class="text-4xl my-4">2019 MS Contest Questions</h3>
                            <iframe src="https://purplecomet.org/files/2019MS_English.pdf" class="w-11/12 h-screen"/>
            <iframe src="https://purplecomet.org/files/2019HS_Spanish.pdf"/>`
	got := PurpleCometPapers(page)
	want := []National{
		{Competition: "purple-comet", Year: 2019, Paper: "hs", URL: "https://purplecomet.org/files/2019HS_English.pdf"},
		{Competition: "purple-comet", Year: 2019, Paper: "ms", URL: "https://purplecomet.org/files/2019MS_English.pdf"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
