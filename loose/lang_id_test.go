package loose

import "testing"

func TestLangIndonesian(t *testing.T) {
	for _, c := range []struct{ text, named, want string }{
		{"Misalkan x, y bilangan asli sehingga 2x + 3y = 2020. Nilai terbesar yang mungkin dari 3x + 2y adalah...", "", "id"},
		{"Misalkan x, y bilangan asli sehingga 2x + 3y = 2020. Nilai terbesar yang mungkin dari 3x + 2y adalah...", "id", "id"},
		{"Let x and y be positive integers with 2x + 3y = 2020. Find the largest value of 3x + 2y.", "id", "en"},
		{"$x^2 + y^2 = 2020$", "id", "id"},
	} {
		if got := Lang(c.text, c.named); got != c.want {
			t.Errorf("Lang(%q, %q) = %q, want %q", c.text, c.named, got, c.want)
		}
	}
}
