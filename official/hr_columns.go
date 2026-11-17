package official

import "slices"

// HrColumnShift is how far down HrColumns sets the right column of a page: below its left column and above the end of any page.
const HrColumnShift = 1 << 18

// HrColumnMid gives the gutter of each page of a paper in two columns, the x that parts its columns: of the x in the middle of the page, between three tenths and seven tenths of the way from the left edge of its leftmost run to the right edge of its rightmost run, the leftmost of those that the fewest runs cross. The columns of a page need not be as wide as each other, nor its rightmost run end at the right margin; the leftmost x is the gutter and not the gap between the heading of a problem of the right column and its text.
func HrColumnMid(items []PDFItem) map[int]int {
	lo, hi := map[int]int{}, map[int]int{}
	for _, it := range items {
		if l, ok := lo[it.Page]; !ok || it.Left < l {
			lo[it.Page] = it.Left
		}
		if r, ok := hi[it.Page]; !ok || it.Right() > r {
			hi[it.Page] = it.Right()
		}
	}
	// cover counts, for each x of a page from its left edge, the runs that start at x or before it and end after it.
	cover := map[int][]int{}
	for pg, l := range lo {
		cover[pg] = make([]int, hi[pg]-l+2)
	}
	for _, it := range items {
		c, l := cover[it.Page], lo[it.Page]
		if it.Right() > it.Left {
			c[it.Left-l]++
			c[it.Right()-l]--
		}
	}
	mid := map[int]int{}
	for pg, l := range lo {
		c, w := cover[pg], hi[pg]-l
		for x := 1; x < len(c); x++ {
			c[x] += c[x-1]
		}
		best := w * 3 / 10
		for x := best; x <= w*7/10; x++ {
			if c[x] < c[best] {
				best = x
			}
		}
		mid[pg] = l + best
	}
	return mid
}

// HrColumns reads a paper in two columns as a paper in one: each run that starts right of the middle of its page moves down by HrColumnShift, so that the right column comes after the left one, and left by the distance between the left edges of the two columns, so that the headings of both columns line up.
func HrColumns(items []PDFItem) []PDFItem {
	mid := HrColumnMid(items)
	left, right := map[int]int{}, map[int]int{}
	for _, it := range items {
		side := left
		if it.Left >= mid[it.Page] {
			side = right
		}
		if l, ok := side[it.Page]; !ok || it.Left < l {
			side[it.Page] = it.Left
		}
	}
	out := slices.Clone(items)
	for i, it := range out {
		if it.Left >= mid[it.Page] {
			out[i].Top += HrColumnShift
			if l, ok := left[it.Page]; ok {
				out[i].Left -= right[it.Page] - l
			}
		}
	}
	return out
}
