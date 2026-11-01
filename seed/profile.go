package seed

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// Row is one seed row: one problem row of a dataset, with its labels and a pointer back to the row.
type Row struct {
	Source string `json:"source"`
	File   string `json:"file"`
	Row    int    `json:"row"`
	ID     string `json:"id,omitempty"`
	// Labels are the competition labels of the row, the most exact first. The locator resolves them through the alias table.
	Labels    []string `json:"labels"`
	Year      int      `json:"year,omitempty"`
	Number    string   `json:"number,omitempty"`
	Country   string   `json:"country,omitempty"`
	Lang      string   `json:"lang,omitempty"`
	Statement string   `json:"statement"`
	Answer    string   `json:"answer,omitempty"`
	Solutions int      `json:"solutions,omitempty"`
	Ref       string   `json:"ref,omitempty"`
	// Author is the id of the register source that wrote the text, when it is not the dataset. For example, the NuminaMath rows from the notes files have the texts of Evan Chen.
	Author string `json:"author,omitempty"`
}

// Profile tells how to read the rows of one dataset.
type Profile struct {
	// Columns are the columns to read. Other columns are not read.
	Columns []string
	// Files tells if a data file of the dataset has problem rows. Nil means all files.
	Files func(path string) bool
	// Map makes a seed row from a record. It returns false for a row that is not a problem of a competition, for example a synthetic row.
	Map func(r Record) (Row, bool)
}

var yearRE = regexp.MustCompile(`(?:^|[^0-9])(19[0-9]{2}|20[0-9]{2})(?:[^0-9]|$)`)

// Year returns the first year from 1900 to 2099 in s, or 0.
func Year(s string) int {
	m := yearRE.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	y, _ := strconv.Atoi(m[1])
	return y
}

var leadYearRE = regexp.MustCompile(`^\s*(19|20)[0-9]{2}(\s*[-/]\s*[0-9]{2,4})?\s+`)

// noYear removes a year at the start of a label: "2023 Germany TST" gives "Germany TST".
func noYear(s string) string { return strings.TrimSpace(leadYearRE.ReplaceAllString(s, "")) }

func count(r Record, cols ...string) int {
	n := 0
	for _, c := range cols {
		for _, v := range r[c] {
			if strings.TrimSpace(v) != "" {
				n++
			}
		}
	}
	return n
}

func labels(ls ...string) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range ls {
		l = strings.TrimSpace(html.UnescapeString(l))
		if l != "" && !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}

// numinaLabels gives the labels of a NuminaMath row. The rows from the IMO Compendium have the exam IMO, but they are the longlist and shortlist problems of the year in the order of the book, so problem 1 of the book is not problem 1 of the IMO. These rows get the label "IMO Compendium".
func numinaLabels(exam, path string) []string {
	if strings.Contains(path, "compendium") {
		return labels("IMO Compendium")
	}
	return labels(exam)
}

// hmmtRE reads the tournament and the round of a NuminaMath row of HMMT from its resource path, such as "en-102-2007-feb-alg-solutions.jsonl" or "en-164-tournaments-2013-hmic-solutions.jsonl".
var hmmtRE = regexp.MustCompile(`[0-9]{4}-(feb|nov|hmic)(?:-([a-z0-9]+))?-solutions\.jsonl$`)

// hmmtTournaments gives the label of each tournament of HMMT in a resource path.
var hmmtTournaments = map[string]string{"feb": "HMMT February", "nov": "HMMT November", "hmic": "HMIC"}

// hmmtRounds gives the letter of the position of each round of HMMT in a resource path. The rounds of one tournament share the numbers 1 to 10, so the letter tells them apart. Algebra (to 2016) and Algebra and Number Theory (from 2017) are never in one year, so they share the letter a. Agent decision: the six rounds of 2011 with two subjects each, such as "algcalc", have no letter, because the order of their problems is not the order of the subject rounds.
var hmmtRounds = map[string]string{
	"feb-alg": "a", "feb-algnt": "a", "feb-geo": "g", "feb-comb": "c", "feb-calc": "k", "feb-adv": "v",
	"feb-gen1": "e", "feb-gen2": "f", "feb-guts": "u", "feb-team": "t", "feb-team1": "t", "feb-team2": "s", "feb-oral": "o",
	"nov-gen": "e", "nov-gen1": "e", "nov-gen2": "f", "nov-thm": "h", "nov-team": "t", "nov-guts": "u",
	"hmic": "p",
}

// hmmtNumberRE reads a problem label of an HMMT row, such as "7", "A7" or "AT10".
var hmmtNumberRE = regexp.MustCompile(`^[A-Z]{0,2}([0-9]{1,2})$`)

// numinaRow gives the labels and the problem label of a NuminaMath row. For a row of HMMT, the label names the tournament and the problem label has the letter of the round, such as "a7" for problem 7 of the algebra round of February. A problem label of HMMT that the round does not explain, such as "M_{13}" in a guts round, gives no problem label.
func numinaRow(exam, path, number string) ([]string, string) {
	m := hmmtRE.FindStringSubmatch(path)
	if m == nil || !strings.HasPrefix(path, "HarvardMIT/") {
		return numinaLabels(exam, path), number
	}
	round := m[1]
	if m[2] != "" {
		round += "-" + m[2]
	}
	letter, ok := hmmtRounds[round]
	n := hmmtNumberRE.FindStringSubmatch(strings.TrimSpace(number))
	if !ok || n == nil {
		return labels(hmmtTournaments[m[1]]), ""
	}
	return labels(hmmtTournaments[m[1]]), letter + n[1]
}

// numinaAuthor gives the author of a NuminaMath row from its resource path. The files that end in "-notes.jsonl" have the texts of the notes of Evan Chen.
func numinaAuthor(path string) string {
	if strings.HasSuffix(path, "-notes.jsonl") {
		return "evanchen"
	}
	return ""
}

// Profiles maps a source id of the source register to the profile of its dataset. A source with no profile uses Generic.
var Profiles = map[string]Profile{
	"hf-mathnet": {
		Columns: []string{"id", "competition", "country", "language", "problem_markdown", "final_answer", "solutions_markdown"},
		Map: func(r Record) (Row, bool) {
			comp := r.Get("competition")
			st := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.Get("problem_markdown")), "Problem:"))
			return Row{
				ID: r.Get("id"), Labels: labels(comp, r.Get("country")+" "+comp), Year: Year(comp),
				Country: r.Get("country"), Lang: r.Get("language"), Statement: st,
				Answer: r.Get("final_answer"), Solutions: count(r, "solutions_markdown"),
			}, st != ""
		},
	},
	"hf-numina-olympiads": {
		Columns: []string{"year", "tier", "problem_label", "exam", "problem", "solution", "metadata.resource_path"},
		Map: func(r Record) (Row, bool) {
			y, _ := strconv.Atoi(r.Get("year"))
			ls, n := numinaRow(r.Get("exam"), r.Get("metadata.resource_path"), r.Get("problem_label"))
			return Row{
				Labels: ls, Year: y, Number: n, Lang: "en",
				Statement: r.Get("problem"), Solutions: count(r, "solution"), Ref: r.Get("metadata.resource_path"),
				Author: numinaAuthor(r.Get("metadata.resource_path")),
			}, r.Get("problem") != ""
		},
	},
	"hf-numina-olympiads-ref": {
		Columns: []string{"problem", "solution", "answer", "metadata.exam", "metadata.year", "metadata.problem_label", "metadata.resource_path"},
		Map: func(r Record) (Row, bool) {
			y, _ := strconv.Atoi(r.Get("metadata.year"))
			ls, n := numinaRow(r.Get("metadata.exam"), r.Get("metadata.resource_path"), r.Get("metadata.problem_label"))
			return Row{
				Labels: ls, Year: y, Number: n, Lang: "en",
				Statement: r.Get("problem"), Answer: r.Get("answer"), Solutions: count(r, "solution"), Ref: r.Get("metadata.resource_path"),
				Author: numinaAuthor(r.Get("metadata.resource_path")),
			}, r.Get("problem") != ""
		},
	},
	"hf-ai-mo-aops": {
		Columns: []string{"problem", "solution", "metadata.path"},
		Map: func(r Record) (Row, bool) {
			path := r.Get("metadata.path")
			parts := strings.Split(path, "/")
			var contest, topic string
			if len(parts) >= 2 {
				contest = parts[len(parts)-2]
				topic = strings.TrimSuffix(parts[len(parts)-1], ".json")
			}
			ref := ""
			if topic != "" {
				ref = "aops-topic:" + topic
			}
			return Row{
				ID: topic, Labels: labels(noYear(contest), contest), Year: Year(contest), Lang: "en",
				Statement: r.Get("problem"), Solutions: count(r, "solution"), Ref: ref,
			}, r.Get("problem") != "" && strings.HasPrefix(path, "Contest Collections")
		},
	},
	"hf-aops-wiki": {
		Columns: []string{"category", "contest", "name", "which", "problem", "source", "solution_1", "solution_2", "solution_3", "solution_4", "solution_5"},
		Map: func(r Record) (Row, bool) {
			name := r.Get("name")
			ref := r.Get("source")
			if strings.HasPrefix(ref, "/") {
				ref = "https://artofproblemsolving.com" + ref
			}
			return Row{
				Labels: labels(noYear(name), r.Get("contest"), name), Year: Year(name), Number: r.Get("which"), Lang: "en",
				Statement: r.Get("problem"), Solutions: count(r, "solution_1", "solution_2", "solution_3", "solution_4", "solution_5"), Ref: ref,
			}, r.Get("problem") != ""
		},
	},
	"hf-omni-math": {
		Columns: []string{"problem", "answer", "solution", "source", "difficulty"},
		Map: func(r Record) (Row, bool) {
			src := r.Get("source")
			return Row{
				Labels: labels(src, strings.ReplaceAll(src, "_", " ")), Year: Year(src), Lang: "en",
				Statement: r.Get("problem"), Answer: r.Get("answer"), Solutions: count(r, "solution"),
			}, r.Get("problem") != ""
		},
	},
	// The TÜBİTAK set has each problem as an image only. The seed row records the year and the subject, and milestone M6 reads the image.
	"hf-tubitak": {
		Columns: []string{"id", "subject", "year", "stage", "question_number"},
		Files:   func(p string) bool { return strings.Contains(p, "matematik") },
		Map: func(r Record) (Row, bool) {
			y, _ := strconv.Atoi(r.Get("year"))
			return Row{ID: r.Get("id"), Labels: labels("TÜBİTAK " + r.Get("subject")), Year: y, Country: "TR", Lang: "tr"}, r.Get("id") != ""
		},
	},
	"hf-olympiadbench": {
		Columns: []string{"id", "question", "final_answer", "solution", "language", "subfield"},
		Files:   func(p string) bool { return strings.Contains(p, "maths") && !strings.Contains(p, "CEE") },
		Map: func(r Record) (Row, bool) {
			return Row{
				ID: r.Get("id"), Labels: labels("OlympiadBench"), Lang: r.Get("language"),
				Statement: r.Get("question"), Answer: strings.Join(r["final_answer"], "; "), Solutions: count(r, "solution"),
			}, r.Get("question") != ""
		},
	},
	"hf-numinamath-1-5": {
		Columns: []string{"problem", "answer", "solution", "source", "synthetic", "problem_is_valid"},
		Map: func(r Record) (Row, bool) {
			if r.Get("synthetic") == "true" {
				return Row{}, false
			}
			src := r.Get("source")
			// Only these sources hold competition problems. The others are textbook and exam sets.
			switch src {
			case "olympiads", "olympiads_ref", "aops_forum", "amc_aime", "cn_contest", "inequalities", "number_theory", "cn_k12_olympiads":
			default:
				return Row{}, false
			}
			return Row{
				Labels: labels("numina:" + src), Lang: "en",
				Statement: r.Get("problem"), Answer: r.Get("answer"), Solutions: count(r, "solution"),
			}, r.Get("problem") != ""
		},
	},
	"hf-amc12-full": {
		Columns: []string{"problem_id", "question", "answer"},
		Map: func(r Record) (Row, bool) {
			m := amcIDRE.FindStringSubmatch(r.Get("problem_id"))
			if m == nil || r.Get("question") == "" {
				return Row{}, false
			}
			y, _ := strconv.Atoi(m[1])
			paper := m[2]
			// The AMC 12 had one paper in 2000 and 2001. The dataset gives it the letter A.
			if y < 2002 {
				paper = ""
			}
			return Row{
				ID: r.Get("problem_id"), Labels: labels("AMC 12"+paper, "AMC 12"), Year: y, Number: m[3], Lang: "en",
				Statement: r.Get("question"), Answer: r.Get("answer"),
			}, true
		},
	},
}

// amcIDRE reads an id of the AMC 12 set, such as 2019A-P15: the year, the paper and the problem.
var amcIDRE = regexp.MustCompile(`^(\d{4})([AB])-P?(\d{1,2})$`)

// The columns that Generic looks for, in order.
var (
	genStatement = []string{"problem", "question", "Problem", "Question", "problem_markdown", "statement", "informal_statement", "informal_problem", "problem_statement", "prompt"}
	genAnswer    = []string{"answer", "final_answer", "Answer", "ground_truth", "expected_answer"}
	genSolution  = []string{"solution", "solutions", "Solution", "proof", "reference_solution", "reasoning"}
	genLabel     = []string{"competition", "contest", "exam", "source", "Source", "origin", "category", "subject", "part"}
	genYear      = []string{"year", "Year"}
	genNumber    = []string{"problem_label", "problem_number", "number", "problem_idx", "No", "index"}
	genID        = []string{"id", "ID", "uid", "problem_id", "unique_id", "idx"}
	genLang      = []string{"language", "lang"}
)

// GenericFiles selects the data files of a dataset that the generic profile reads. Putnam-AXIOM has the original problems, versions with a changed answer form and variations made by a program. Only the originals are human text.
var GenericFiles = map[string]func(path string) bool{
	"hf-putnam-axiom": func(p string) bool { return strings.Contains(p, "originals") },
}

// Generic makes a profile from the column names of a table. It serves the small human sets, which have the usual names.
func Generic(cols []string) Profile {
	has := map[string]bool{}
	for _, c := range cols {
		has[c] = true
	}
	var pick []string
	add := func(list []string) {
		for _, c := range list {
			if has[c] {
				pick = append(pick, c)
			}
		}
	}
	for _, l := range [][]string{genStatement, genAnswer, genSolution, genLabel, genYear, genNumber, genID, genLang} {
		add(l)
	}
	return Profile{
		Columns: pick,
		Map: func(r Record) (Row, bool) {
			st := r.First(genStatement...)
			id, num := r.First(genID...), r.First(genNumber...)
			if num == "" && Position(id) != "" {
				// Some sets give the position as the id, as in "A-1".
				num = id
			}
			var ls []string
			for _, c := range genLabel {
				ls = append(ls, r.Get(c))
			}
			y, _ := strconv.Atoi(r.First(genYear...))
			if y == 0 {
				for _, l := range ls {
					if y = Year(l); y != 0 {
						break
					}
				}
			}
			return Row{
				ID: id, Labels: labels(ls...), Year: y, Number: num, Lang: r.First(genLang...),
				Statement: st, Answer: r.First(genAnswer...), Solutions: count(r, genSolution...),
			}, st != ""
		},
	}
}
