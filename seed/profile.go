package seed

import (
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
		l = strings.TrimSpace(l)
		if l != "" && !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
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
			return Row{
				Labels: labels(r.Get("exam")), Year: y, Number: r.Get("problem_label"), Lang: "en",
				Statement: r.Get("problem"), Solutions: count(r, "solution"), Ref: r.Get("metadata.resource_path"),
			}, r.Get("problem") != ""
		},
	},
	"hf-numina-olympiads-ref": {
		Columns: []string{"problem", "solution", "answer", "metadata.exam", "metadata.year", "metadata.problem_label", "metadata.resource_path"},
		Map: func(r Record) (Row, bool) {
			y, _ := strconv.Atoi(r.Get("metadata.year"))
			return Row{
				Labels: labels(r.Get("metadata.exam")), Year: y, Number: r.Get("metadata.problem_label"), Lang: "en",
				Statement: r.Get("problem"), Answer: r.Get("answer"), Solutions: count(r, "solution"), Ref: r.Get("metadata.resource_path"),
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
}

// The columns that Generic looks for, in order.
var (
	genStatement = []string{"problem", "question", "Problem", "Question", "problem_markdown", "statement", "informal_statement", "informal_problem", "prompt"}
	genAnswer    = []string{"answer", "final_answer", "Answer", "ground_truth", "expected_answer"}
	genSolution  = []string{"solution", "solutions", "Solution", "proof", "reference_solution"}
	genLabel     = []string{"competition", "contest", "exam", "source", "Source", "origin", "category", "subject"}
	genYear      = []string{"year", "Year"}
	genNumber    = []string{"problem_label", "problem_number", "number", "problem_idx", "No"}
	genID        = []string{"id", "ID", "uid", "problem_id", "unique_id", "idx"}
	genLang      = []string{"language", "lang"}
)

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
				ID: r.First(genID...), Labels: labels(ls...), Year: y, Number: r.First(genNumber...), Lang: r.First(genLang...),
				Statement: st, Answer: r.First(genAnswer...), Solutions: count(r, genSolution...),
			}, st != ""
		},
	}
}
