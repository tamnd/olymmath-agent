package ocr

import "testing"

func TestSplitCapitalWord(t *testing.T) {
	pages := []Page{{N: 1, Text: "XXII OLIMPÍADA de MAYO\nPrimer Nivel\nPROBLEMA 1\nEn una hoja están escritos siete números enteros positivos diferentes.\nPROBLEMA 2\nEn una competición deportiva solo participan tres atletas."}}
	got := Split(pages)
	if len(got) != 2 || got[0].Number != 1 || got[1].Number != 2 {
		t.Fatalf("Split gave %+v", got)
	}
	if got[0].Text != "En una hoja están escritos siete números enteros positivos diferentes." {
		t.Errorf("p1 = %q", got[0].Text)
	}
}
