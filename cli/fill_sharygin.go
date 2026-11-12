package cli

import (
	"context"
	"strconv"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	// The year node of problems.ru holds the final round in a child node, whose name is not "Заключительный этап", so the lane reads the whole year node.
	ruLanes["sharygin"] = ruLane{source: "geometry-ru", node: prSharygin, read: sharyginRead, holds: prSameYear}
}

// prSharygin is the node of the Sharygin olympiad in the source catalogue of problems.ru.
const prSharygin = 206102

// sharyginFinals are the papers in Russian of the final round of the Sharygin olympiad on geometry.ru, by year. The papers of 2024 and 2025 have the solutions after each problem.
var sharyginFinals = map[int]string{
	2013: "https://geometry.ru/olimp/2013/usl.pdf",
	2017: "https://geometry.ru/olimp/2017/usl_final.pdf",
	2019: "https://geometry.ru/olimp/2019/final_problems.pdf",
	2021: "https://geometry.ru/olimp/2021/final.pdf",
	2022: "https://geometry.ru/olimp/2022/final_rus_2022.pdf",
	2023: "https://geometry.ru/olimp/2023/final_rus_2023.pdf",
	2024: "https://geometry.ru/olimp/2024/2024_final_sol_rus.pdf",
	2025: "https://geometry.ru/olimp/2025/2025_final_sol_rus.pdf",
}

// sharyginRead reads the problems of the final round of a year of the Sharygin olympiad from its paper in Russian.
func sharyginRead(ctx context.Context, fc *fetch.Client, source, edition string) ([]official.RuProblem, error) {
	y, err := strconv.Atoi(edition)
	if err != nil {
		return nil, err
	}
	u, ok := sharyginFinals[y]
	if !ok {
		return nil, nil
	}
	pages, err := pdfPages(ctx, fc, source, u)
	if err != nil {
		return nil, err
	}
	return official.SharyginFinal(pages, edition, u), nil
}
