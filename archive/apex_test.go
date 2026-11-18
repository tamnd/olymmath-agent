package archive

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"os"
	"strconv"
	"strings"
	"testing"
)

// apexFixture reads the records of testdata/apex.jsonl: full records of the site as captured on 2026-11-16, one for each line.
func apexFixture(t *testing.T) map[string]ApexRecord {
	t.Helper()
	b, err := os.ReadFile("testdata/apex.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]ApexRecord{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		r, err := ParseApex(sc.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		out[r.Slug] = r
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestApexPlace(t *testing.T) {
	recs := apexFixture(t)
	if len(recs) < 27 {
		t.Fatalf("%d records in the fixture", len(recs))
	}
	for slug, want := range map[string]string{
		// The district stage of 2001/02 was written in November or December 2001.
		"ukr_2002_03697c": "ua-umo-district/2001/p93",
		"ukr_2010_69c876": "ua-umo-district/2009/p64",
		"ukr_2001_02d699": "ua-umo/2001/p86",
		"ukr_2008_3ecc24": "ua-umo/2008/p84",
		"ukr_2008_40c970": "ua-umo/2008/p86",
		// The Kyiv city olympiad of 2018/19, first and second round.
		"ukr_2019_1a6514": "ua-kyiv/2019/a84",
		"ukr_2019_303382": "ua-kyiv/2019/b71",
		"ukr_2007_189568": "ua-kyiv/2007/p72",
		// The Kyiv district olympiad of 2006/07.
		"ukr_2007_077948": "ua-umo-district/2006/p61",
		// The selection of the Kyiv team of 2006/07, round II, and of 2022/23, the additional round V.
		"ukr_2007_044962": "ua-kyiv-selection/2007/b83",
		"ukr_2023_4bde96": "ua-kyiv-selection/2023/e91",
		"ukr_2009_1b9dc0": "ua-tst/2009/b1",
		"ukr_2016_94b6d2": "ua-intl-selection/2016/i3",
		"ukr_2022_53ce3a": "ua-intl-selection/2021/p5",
		"ukr_2014_a613d5": "ua-thematic/2014/a93",
	} {
		r, ok := recs[slug]
		if !ok {
			t.Fatalf("no record %s", slug)
		}
		code, year, number, why := ApexPlace(r)
		if why != "" {
			t.Errorf("%s: %s", slug, why)
			continue
		}
		if got := code + "/" + strconv.Itoa(year) + "/" + number; got != want {
			t.Errorf("%s: got %s, want %s", slug, got, want)
		}
	}
	// A variant and a training problem have no place.
	for _, slug := range []string{"ukr_2009_113e24", "ukr_2007_17f7fe"} {
		if code, _, number, why := ApexPlace(recs[slug]); why == "" {
			t.Errorf("%s: placed at %s %s", slug, code, number)
		}
	}
	for _, r := range recs {
		code, _, number, why := ApexPlace(r)
		if why == "" && (code == "" || strings.Trim(number[:1], "abcdegijnp") != "") {
			t.Errorf("%s: placed at %s %s", r.Slug, code, number)
		}
	}
}

func TestApexPlaceOther(t *testing.T) {
	for _, c := range []struct {
		r   ApexRecord
		why bool
	}{
		{ApexRecord{Competition: "KyivMathBattle", Level: "regional", School: "2018-2019", Ref: "8.1"}, true},
		{ApexRecord{Competition: "KyivSelection", Level: "regional", School: "2008-2009", Ref: "8.1", SubEvent: "III тур"}, false},
		{ApexRecord{Competition: "KyivCityMO", Level: "regional", School: "2020-2021", Ref: "8.1", SubEvent: "II етап"}, true},
		{ApexRecord{Competition: "KyivCityMO", Level: "regional", School: "2006-2007", Ref: "8.1"}, false},
		{ApexRecord{Competition: "UkrMO", Level: "regional", School: "2023-2024", Ref: "7.1", SubEvent: "Основний тур відбору команди Києва", Round: "III етап — LXXIX Київська міська олімпіада юних математиків"}, true},
		{ApexRecord{Competition: "KyivDistrictMO", Level: "district", School: "2006-2007", Ref: "6.1"}, false},
		{ApexRecord{Competition: "KyivDistrictMO", Level: "district", School: "2006-2007", Ref: "тренувальна задача 6"}, true},
		{ApexRecord{Competition: "UkrMO", Level: "national", School: "2018", Ref: "8.1"}, true},
		{ApexRecord{Competition: "UkrMO", Level: "national", School: "2017-2018", Ref: "8"}, true},
		{ApexRecord{Competition: "UkrMO", Level: "regional", School: "2017-2018", Ref: "8.1", Round: "III (обласний) етап Всеукраїнської математичної олімпіади — LXXIII Київська міська олімпіада юних математиків"}, false},
	} {
		code, _, _, why := ApexPlace(c.r)
		if (why != "") != c.why {
			t.Errorf("%+v: code %q, reason %q", c.r, code, why)
		}
	}
	code, year, _, _ := ApexPlace(ApexRecord{Competition: "UkrMO", Level: "regional", School: "2017-2018", Ref: "8.1", Round: "III етап — LXXIII Київська міська олімпіада юних математиків"})
	if code != "ua-kyiv" || year != 2018 {
		t.Errorf("Kyiv: %s %d", code, year)
	}
	kyiv := "III етап — LXIV Київська міська олімпіада юних математиків"
	for _, c := range []struct{ level, ref, sub, want string }{
		{"regional", "7.1", "I тур", "ua-kyiv a71"},
		{"regional", "7.1", "II тур", "ua-kyiv b71"},
		{"regional", "7.1", "І тур", "ua-kyiv a71"}, // a Cyrillic І
		{"regional", "II тур · 8.3", "", "ua-kyiv b83"},
		{"regional", "8.2.1", "I тур", ""},
		{"regional", "I тур · 9.4.1", "", ""},
		{"regional", "I тур · 9.4", "II тур", ""},
		{"national", "II тур · 9.8", "", "ua-umo p98"},
		{"national", "9.6", "Другий день", "ua-umo p96"},
	} {
		code, _, number, why := ApexPlace(ApexRecord{Competition: "UkrMO", Level: c.level, School: "2008-2009", Ref: c.ref, SubEvent: c.sub, Round: kyiv})
		got := code + " " + number
		if why != "" {
			got = ""
		}
		if got != c.want {
			t.Errorf("%s %q %q: got %q (%s), want %q", c.level, c.ref, c.sub, got, why, c.want)
		}
	}
}

func TestApexText(t *testing.T) {
	r := apexFixture(t)["ukr_2002_03697c"]
	text := ApexText(r)
	if !strings.Contains(text, "![](https://apex-minds.org/images/0ffe159c-a63a-4162-8d2e-068b09d1d697.png)") || strings.Contains(text, "](images/") {
		t.Errorf("figure link: %q", text)
	}
	name, ok := IsApexImage("https://apex-minds.org/images/0ffe159c-a63a-4162-8d2e-068b09d1d697.png")
	if !ok || name != "0ffe159c-a63a-4162-8d2e-068b09d1d697.png" {
		t.Errorf("IsApexImage: %q %v", name, ok)
	}
	if _, ok := IsApexImage("p1.fig1.png"); ok {
		t.Error("a local figure is not a figure of the site")
	}
	if got := ApexAnswer(r); got != "$5000$." {
		t.Errorf("answer %q", got)
	}
}

func TestApexGzipAndSlugs(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"slug":"ukr_2002_03697c","competition_code":"UkrMO"}`))
	_ = zw.Close()
	r, err := ParseApex(buf.Bytes())
	if err != nil || r.Slug != "ukr_2002_03697c" {
		t.Fatalf("%v %+v", err, r)
	}
	manifest := `{"schema":["g","s","y","c"],"rows":[["btw_1990_08338e","baltic-way",1990,"BalticWay"],["ukr_2002_03697c","ukraine",2002,"UkrMO"],["kyb_2019_aaaaaa","kyiv",2019,"KyivMathBattle"]]}`
	slugs, err := ApexSlugs([]byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(slugs, ",") != "ukr_2002_03697c,kyb_2019_aaaaaa" {
		t.Errorf("slugs %v", slugs)
	}
}
