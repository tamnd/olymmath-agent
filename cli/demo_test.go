package cli

import "testing"

func TestDemoPGM(t *testing.T) {
	w, h, pix, ok := demoPGM([]byte("P5\n3 2\n255\n\x00\x10\x20\x30\x40\x50"))
	if !ok || w != 3 || h != 2 || len(pix) != 6 || pix[5] != 0x50 {
		t.Fatalf("demoPGM = %d, %d, %v, %v", w, h, pix, ok)
	}
	for _, b := range []string{"P6\n3 2\n255\n123456", "P5\n3 2\n255\n12", "P5\n3 2\n65535\n123456", "P5"} {
		if _, _, _, ok := demoPGM([]byte(b)); ok {
			t.Errorf("demoPGM(%q) reads", b)
		}
	}
}
