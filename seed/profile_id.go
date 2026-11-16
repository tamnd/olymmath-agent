package seed

import "regexp"

// placeholderRE finds the marks that the Indonesian set puts where a paper has a figure or a table, such as [Gambar segitiga ABC] or [Tabel Data KK dan Pemilih per RW]. A statement with such a mark is not the text of the paper.
var placeholderRE = regexp.MustCompile(`\[(?:Gambar|Tabel|Diagram|Grafik|Grid|Piramida|Pola)\b[^\]]*\]`)

func init() {
	// The Indonesian olympiad set (hafidhsoekma/math-olympiad-indonesian-benchmark) has the columns question, final_answer and source, as osn_kabupaten_sd_2023, and its statements are in Indonesian. Its card gives no licence.
	g := Generic([]string{"question", "final_answer", "source"})
	Profiles["hf-indonesian-olympiad"] = Profile{
		Columns: g.Columns,
		Map: func(r Record) (Row, bool) {
			row, ok := g.Map(r)
			row.Lang = "id"
			return row, ok && !placeholderRE.MatchString(row.Statement)
		},
	}
}
