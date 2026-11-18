package archive

import (
	"strconv"
	"testing"
)

func TestApexMore(t *testing.T) {
	for _, c := range []struct {
		r    ApexRecord
		want string
	}{
		// The Kyiv selection numbers its rounds I to V; an additional round keeps its numeral.
		{ApexRecord{Competition: "KyivSelection", School: "2022-2023", SubEvent: "V тур (додатковий)", Ref: "9.1"}, "ua-kyiv-selection/2023/e91"},
		{ApexRecord{Competition: "KyivSelection", School: "2014-2015", SubEvent: "IV (додатковий) тур", Ref: "8.2"}, "ua-kyiv-selection/2015/d82"},
		{ApexRecord{Competition: "KyivSelection", School: "2009-2010", Ref: "І тур · 8.4"}, "ua-kyiv-selection/2010/a84"},
		{ApexRecord{Competition: "KyivSelection", School: "2008-2009", SubEvent: "III тур", Ref: "8.1"}, "ua-kyiv-selection/2009/c81"},
		{ApexRecord{Competition: "ZakarpattiaSelection", School: "2023-2024", Ref: "8.1"}, "ua-zakarpattia-selection/2024/p81"},
		// The camps of 2008 and 2009 number each round from 1, the selection of 2010 on through the rounds.
		{ApexRecord{Competition: "UkrTST", School: "2008-2009", Round: "Відбірково-тренувальні збори кандидатів до команди України (2009)", SubEvent: "II тур", Ref: "1"}, "ua-tst/2009/b1"},
		{ApexRecord{Competition: "UkrTST", School: "2011-2012", Round: "Відбір команди України на 53-тю Міжнародну математичну олімпіаду", SubEvent: "IV тур", Ref: "10"}, "ua-tst/2012/p10"},
		{ApexRecord{Competition: "UkrTST", School: "2002-2003", Round: "Відбірково-тренувальні збори кандидатів до команди України", Ref: "5"}, "ua-tst/2003/p5"},
		// The groups of a day of the international selection.
		{ApexRecord{Competition: "IntlTeamSelection", School: "2015-2016", Round: "Відбір на Міжнародні математичні олімпіади (2016)", SubEvent: "Перший день — група IMO", Ref: "3"}, "ua-intl-selection/2016/i3"},
		{ApexRecord{Competition: "IntlTeamSelection", School: "2015-2016", Round: "Відбір на Міжнародні математичні олімпіади (2016)", SubEvent: "Перший день — молодша група", Ref: "3"}, "ua-intl-selection/2016/j3"},
		{ApexRecord{Competition: "IntlTeamSelection", School: "2018-2019", Round: "Відбір на міжнародні змагання (2019)", SubEvent: "Третій день — група IMO", Ref: "11.2"}, "ua-intl-selection/2019/i11"},
		{ApexRecord{Competition: "IntlTeamSelection", School: "2018-2019", Round: "Відбір на міжнародні змагання (2019)", SubEvent: "Перший день", Ref: "2"}, "ua-intl-selection/2019/p2"},
		// From the autumn of 2021 the name gives the first year of the school year.
		{ApexRecord{Competition: "IntlTeamSelection", School: "2021-2022", Round: "Відбір на міжнародні змагання (2021)", SubEvent: "Другий день", Ref: "5"}, "ua-intl-selection/2021/p5"},
		{ApexRecord{Competition: "IntlTeamSelection", School: "2024-2025", Round: "Міжнародні математичні змагання за участю школярів України", SubEvent: "Додатковий тур відбору на EGMO", Ref: "2"}, "ua-intl-selection/2025/e2"},
		{ApexRecord{Competition: "ThematicMO", School: "2015-2016", Round: "Олімпіада з алгебри", Ref: "алгебра · 9.3"}, "ua-thematic/2016/a93"},
		{ApexRecord{Competition: "ThematicMO", School: "2015-2016", Round: "Тематичні олімпіади — Олімпіада з теорії чисел", Ref: "10.2"}, "ua-thematic/2016/n102"},
		{ApexRecord{Competition: "SETUniversityMO", School: "2021-2022", Ref: "II тур · 9.3"}, "ua-set-university/2022/b93"},
		{ApexRecord{Competition: "NewYearZanzibar", School: "2022-2023", Ref: "4.1"}, "ua-newyear-zanzibar/2023/p41"},
	} {
		code, year, number, why := ApexPlace(c.r)
		if got := code + "/" + strconv.Itoa(year) + "/" + number; why != "" || got != c.want {
			t.Errorf("%+v: got %s (%s), want %s", c.r, got, why, c.want)
		}
	}
	for _, r := range []ApexRecord{
		{Competition: "KyivMathBattle", School: "2018-2019"},
		{Competition: "RichelieuMO", School: "2000-2001"},
		{Competition: "KyivSelection", School: "2020-2021", Round: "Відбір на Київські гуртки (2020)", Ref: "8.4"},
		{Competition: "KyivSelection", School: "2024-2025", Ref: "11.1"},
		{Competition: "KyivSelection", School: "2010-2011", SubEvent: "II тур", Ref: "8.2.1"},
		{Competition: "IntlTeamSelection", School: "2020-2021", Round: "Відбір на міжнародні змагання (2020)", SubEvent: "Другий день", Ref: "5"},
		{Competition: "ThematicMO", School: "2015-2016", Round: "Тематичні олімпіади з математики", Ref: "9.1"},
		{Competition: "SETUniversityMO", School: "2021-2022", Ref: "I тур · 7"},
	} {
		if code, year, number, why := ApexPlace(r); why == "" {
			t.Errorf("%+v: placed at %s/%d/%s", r, code, year, number)
		}
	}
}
