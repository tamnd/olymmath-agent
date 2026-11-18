package official

import (
	"reflect"
	"testing"
)

func TestOMARoundPapers(t *testing.T) {
	page := `{"items":[
{"competition":"oma","year":2025,"visible":true,"files":[
 {"label":"Intercolegial","url":"/contents/enunciados/pdf/oma/2025/oma_inter_2025.pdf"},
 {"label":"Nacional","url":"/contents/enunciados/pdf/oma/2025/oma_nacional_2025.pdf"},
 {"label":"Nacional","url":"/contents/enunciados/pdf/oma/2025/oma_nacional_2025.pdf"},
 {"label":"Soluciones","url":"/contents/enunciados/pdf/oma/2025/sol.pdf"}]},
{"competition":"nandu","year":2024,"visible":true,"files":[
 {"label":"Interescolar","url":"/contents/enunciados/pdf/nandu/2024/inter_n_2024.pdf"}]},
{"competition":"nandu","year":2020,"visible":false,"files":[
 {"label":"Zonal","url":"/contents/enunciados/pdf/nandu/2020/zonal_n_2020.pdf"}]},
{"competition":"mayo","year":2024,"files":[
 {"label":"Primer Nivel","url":"/contents/enunciados/pdf/mayo/2024/m1.pdf"}]}]}`
	inter := "https://www.oma.org.ar/contents/enunciados/pdf/oma/2025/oma_inter_2025.pdf"
	nac := "https://www.oma.org.ar/contents/enunciados/pdf/oma/2025/oma_nacional_2025.pdf"
	nandu := "https://www.oma.org.ar/contents/enunciados/pdf/nandu/2024/inter_n_2024.pdf"
	want := []National{
		{Competition: "ar-oma", Year: 2025, Paper: "i1", URL: inter},
		{Competition: "ar-oma", Year: 2025, Paper: "i2", URL: inter},
		{Competition: "ar-oma", Year: 2025, Paper: "i3", URL: inter},
		{Competition: "ar-oma", Year: 2025, Paper: "n1", URL: nac},
		{Competition: "ar-oma", Year: 2025, Paper: "n2", URL: nac},
		{Competition: "ar-oma", Year: 2025, Paper: "n3", URL: nac},
		{Competition: "ar-nandu", Year: 2024, Paper: "i1", URL: nandu},
		{Competition: "ar-nandu", Year: 2024, Paper: "i2", URL: nandu},
		{Competition: "ar-nandu", Year: 2024, Paper: "i3", URL: nandu},
	}
	if got := OMARoundPapers(page); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if got := len(OMAAllPapers(page)); got != len(want)+1 {
		t.Errorf("OMAAllPapers gave %d papers, want %d", got, len(want)+1)
	}
	if got := OMARoundPapers("<html>"); got != nil {
		t.Errorf("not JSON: got %+v", got)
	}
}
