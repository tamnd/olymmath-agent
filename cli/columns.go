package cli

import (
	"context"
	"strconv"

	"github.com/tamnd/olymmath-agent/official"
)

// columnInk is the DemoInk of a paper in two columns that official.HrColumns reads: the rows of the right column of a page start at official.HrColumnShift, and the ink of each column counts apart from the ink of the other.
func columnInk(ctx context.Context, file string, items []official.PDFItem) official.DemoInk {
	mid := official.HrColumnMid(items)
	boxes := map[int][]official.PDFItem{}
	for _, it := range items {
		boxes[it.Page] = append(boxes[it.Page], it)
	}
	rows := map[int][2][]int{}
	return func(page, top, bottom int) bool {
		r, ok := rows[page]
		if !ok {
			r = columnStray(ctx, file, page, boxes[page], mid[page])
			rows[page] = r
		}
		if r[0] == nil {
			return true
		}
		count := func(col, from, to int) int {
			n := 0
			for y := max(from, 0); y < min(to, len(r[col])); y++ {
				n += r[col][y]
			}
			return n
		}
		const s = official.HrColumnShift
		n := 0
		switch {
		case top >= s:
			n = count(1, top-s, bottom-s)
		case bottom > s:
			n = count(0, top, s) + count(1, 0, bottom-s)
		default:
			n = count(0, top, bottom)
		}
		return n >= demoInkMin
	}
}

// columnStray counts the dark pixels of each row of a page outside its runs of text, as demoStray does, in its left column, left of mid, and in its right column apart. A row dark over half the width of its column is a rule, not a drawing.
func columnStray(ctx context.Context, file string, page int, items []official.PDFItem, mid int) [2][]int {
	p := strconv.Itoa(page)
	b, err := figureRunner.Run(ctx, "pdftoppm", "-r", strconv.Itoa(demoDPI), "-gray", "-f", p, "-l", p, "-singlefile", file)
	if err != nil {
		return [2][]int{}
	}
	w, h, pix, ok := demoPGM(b)
	if !ok {
		return [2][]int{}
	}
	mask := make([]bool, w*h)
	const pad = 3
	for _, it := range items {
		for y := max(it.Top-pad, 0); y < min(it.Top+it.Height+pad, h); y++ {
			for x := max(it.Left-pad, 0); x < min(it.Right()+pad, w); x++ {
				mask[y*w+x] = true
			}
		}
	}
	mid = min(max(mid, 0), w)
	out := [2][]int{make([]int, h), make([]int, h)}
	for i, g := range pix {
		if g < 160 && !mask[i] {
			col := 0
			if i%w >= mid {
				col = 1
			}
			out[col][i/w]++
		}
	}
	for y := range h {
		if out[0][y] > mid/2 {
			out[0][y] = 0
		}
		if out[1][y] > (w-mid)/2 {
			out[1][y] = 0
		}
	}
	return out
}
