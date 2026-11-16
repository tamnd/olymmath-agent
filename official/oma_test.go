package official

import (
	"reflect"
	"testing"
)

func TestOMAPapers(t *testing.T) {
	page := `{"updatedAt":"2026-11-01","items":[
{"id":"mayo-2024","competition":"mayo","year":2024,"visible":true,"files":[
 {"label":"Primer Nivel","url":"/contents/enunciados/pdf/mayo/2024/enunciados_MAYO_nivel1_2024.pdf"},
 {"label":"Segundo Nivel","url":"/contents/enunciados/pdf/mayo/2024/enunciados_MAYO_nivel2_2024.pdf"}]},
{"id":"mayo-2021","competition":"mayo","year":2021,"visible":true,"files":[
 {"label":"XXVII Olimpíada de Mayo","url":"/contents/enunciados/pdf/mayo/2021/enunciados_mayo_2021.pdf"}]},
{"id":"mayo-2020","competition":"mayo","year":2020,"visible":false,"files":[
 {"label":"XXVI Olimpíada de Mayo","url":"/contents/enunciados/pdf/mayo/2020/Mayo_2020.pdf"}]},
{"id":"oma-2025","competition":"oma","year":2025,"visible":true,"files":[
 {"label":"Nacional","url":"/contents/enunciados/pdf/oma/2025/oma_nacional_2025.pdf"}]}]}`
	want := []National{
		{Competition: "may-olympiad", Year: 2024, Paper: "level1", URL: "https://www.oma.org.ar/contents/enunciados/pdf/mayo/2024/enunciados_MAYO_nivel1_2024.pdf"},
		{Competition: "may-olympiad", Year: 2024, Paper: "level2", URL: "https://www.oma.org.ar/contents/enunciados/pdf/mayo/2024/enunciados_MAYO_nivel2_2024.pdf"},
		{Competition: "may-olympiad", Year: 2021, URL: "https://www.oma.org.ar/contents/enunciados/pdf/mayo/2021/enunciados_mayo_2021.pdf"},
	}
	if got := OMAPapers(page); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if got := OMAPapers("<html>"); got != nil {
		t.Errorf("not JSON: got %+v", got)
	}
}
