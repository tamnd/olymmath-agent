package seed

import "testing"

func TestIndonesianProfile(t *testing.T) {
	p := Profiles["hf-indonesian-olympiad"]
	row, ok := p.Map(Record{"question": {"Banyaknya tripel bilangan bulat (x, y, z) dengan 0 ≤ x ≤ y ≤ z adalah ...."}, "final_answer": {"153"}, "source": {"osn_kabupaten_smp_2021"}})
	if !ok || row.Lang != "id" || row.Year != 2021 || row.Answer != "153" || len(row.Labels) == 0 {
		t.Errorf("got %+v, %v", row, ok)
	}
	if _, ok := p.Map(Record{"question": {"Perhatikan [Gambar segitiga ABC]. Luas daerah yang diarsir adalah ...."}, "source": {"osn_kabupaten_sd_2023"}}); ok {
		t.Error("a statement with a figure mark was kept")
	}
}
