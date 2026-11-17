package official

import (
	"regexp"
	"testing"
)

func TestHrColumns(t *testing.T) {
	// A page in landscape with a wide left column and a narrower right one, which ends well before the right margin: the gutter lies between them, not at the middle of the runs.
	items := []PDFItem{
		{Page: 1, Top: 100, Left: 31, Width: 15, Height: 13, Text: "1)"},
		{Page: 1, Top: 100, Left: 61, Width: 537, Height: 13, Text: "Fino al 2013, nella colonia penale"},
		{Page: 1, Top: 120, Left: 61, Width: 500, Height: 13, Text: "costituita per il 60% da androidi"},
		{Page: 1, Top: 110, Left: 613, Width: 15, Height: 13, Text: "9)"},
		{Page: 1, Top: 110, Left: 643, Width: 587, Height: 13, Text: "Alberto va in cartoleria"},
		{Page: 2, Top: 50, Left: 31, Width: 24, Height: 13, Text: "12)"},
	}
	mid := HrColumnMid(items)
	if m := mid[1]; m < 598 || m > 613 {
		t.Errorf("HrColumnMid page 1 = %d, want the gutter between 598 and 613", m)
	}
	got := HrColumns(items)
	for i, want := range []struct{ top, left int }{{100, 31}, {100, 61}, {120, 61}, {110 + HrColumnShift, 31}, {110 + HrColumnShift, 61}, {50, 31}} {
		if got[i].Top != want.top || got[i].Left != want.left {
			t.Errorf("HrColumns item %d (%q) at %d,%d, want %d,%d", i, got[i].Text, got[i].Top, got[i].Left, want.top, want.left)
		}
	}
	if items[3].Top != 110 {
		t.Error("HrColumns changed its items")
	}
}

func TestHrMainSize(t *testing.T) {
	// The text of a paper set in Computer Modern Roman: pxMainSize, which takes CMR for a font of formulas, gives the size of the title.
	items := []PDFItem{
		{Size: 22, Family: "QGKUIO+CMBX12", Text: "XXXI Olimpiade Italiana di Matematica"},
		{Size: 15, Family: "RGVVBE+CMR10", Text: "Sia dato un parallelepipedo rettangolo, dove la faccia inferiore"},
		{Size: 15, Family: "RUHRYP+CMMI10", Text: "ABCDA ABCDA ABCDA ABCDA ABCDA ABCDA ABCDA ABCDA ABCDA"},
	}
	if got := pxMainSize(items); got != 22 {
		t.Errorf("pxMainSize = %d, want 22", got)
	}
	if got := hrMainSize(items, hrFonts.math); got != 15 {
		t.Errorf("hrMainSize = %d, want 15", got)
	}
	if got := hrMainSize(items, regexp.MustCompile(`CM`)); got != 0 {
		t.Errorf("hrMainSize of math only = %d, want 0", got)
	}
}

func TestHrJoinLetters(t *testing.T) {
	items := []PDFItem{
		{Page: 1, Top: 10, Left: 10, Width: 20, Text: "pi`"},
		{Page: 1, Top: 10, Left: 30, Width: 30, Text: "u vicina, cos`ı `e"},
		{Page: 1, Top: 10, Left: 70, Width: 30, Text: "finch´e perch´ e `E"},
		{Page: 1, Top: 40, Left: 10, Width: 20, Text: "a`"},
		{Page: 1, Top: 60, Left: 10, Width: 20, Text: "o"},
	}
	got := hrJoinLetters(items, HrAccentsIt)
	var texts []string
	for _, it := range got {
		texts = append(texts, it.Text)
	}
	want := []string{"più vicina, così è", "finché perché È", "a`", "o"}
	if len(texts) != len(want) {
		t.Fatalf("hrJoinLetters = %q, want %q", texts, want)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Errorf("hrJoinLetters = %q, want %q", texts, want)
			break
		}
	}
	if got[0].Width != 50 {
		t.Errorf("the joined item is %d wide, want 50", got[0].Width)
	}
}
