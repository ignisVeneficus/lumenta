package grid

import (
	"fmt"

	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

const (
	WeightHoles  = 10
	WeightBelow  = 5
	WeightUneven = 1
)

type Skyline struct {
	W       int
	Heights []int
}

func NewSkyline(w int) Skyline {
	return Skyline{
		W:       w,
		Heights: make([]int, w),
	}
}

func maxInRange(h []int, x, w int) int {
	max := 0
	for i := x; i < x+w; i++ {
		if h[i] > max {
			max = h[i]
		}
	}
	return max
}

func countHolesCreated(h []int, x, y int, tile *gridData.Rect) int {
	holes := 0
	for i := x; i < x+tile.W; i++ {
		if h[i] < y {
			holes += y - h[i]
		}
	}
	return holes
}
func unevenness(h []int) int {
	sum := 0
	for _, v := range h {
		sum += v
	}
	avg := float64(sum) / float64(len(h))

	u := 0
	for _, v := range h {
		d := int(float64(v) - avg)
		if d < 0 {
			d = -d
		}
		u += d
	}
	return u
}

func healNarrowGaps(h []int, minW int) []int {
	if minW <= 1 {
		// width < 1 never happens; minW==1 means nothing is "too narrow"
		return h
	}
	w := len(h)
	y := 0

	for {
		runStart := -1
		emptyCount := 0

		for x := 0; x < w; x++ {
			filled := h[x] > y
			if !filled {
				emptyCount++
				if runStart == -1 {
					runStart = x
				}
				continue
			}
			// close an empty run
			if runStart != -1 {
				runWidth := x - runStart
				if runWidth < minW {
					for i := runStart; i < x; i++ {
						h[i] = y + 1
					}
				}
				runStart = -1
			}
		}

		// close trailing run (if row ended with empties)
		if runStart != -1 {
			runWidth := w - runStart
			if runWidth < minW {
				for i := runStart; i < w; i++ {
					h[i] = y + 1
				}
			}
		}

		// stop condition: entire row is empty => y is above tallest column
		if emptyCount == w {
			return h
		}

		y++
	}

}
func printOut(s []int) {
	y := 0
	space := 0
	for space < len(s) {
		space = 0
		for i := range s {
			if s[i] > y {
				fmt.Print("█")
			} else {
				fmt.Print(" ")
				space++
			}
		}
		fmt.Print("\n")
		y++
	}
}

func Place(s Skyline, tile *gridData.Rect, minRemainingWidth int) Skyline {
	bestCost := int(^uint(0) >> 1) // max int
	bestX := 0
	bestY := 0
	/*
		fmt.Println("===================================")
		fmt.Printf("[w: %d, h: %d]\n", tile.W, tile.H)
	*/
	for cx := 0; cx <= s.W-tile.W; cx++ {
		cy := maxInRange(s.Heights, cx, tile.W)

		holes := countHolesCreated(s.Heights, cx, cy, tile)

		tmp := make([]int, len(s.Heights))
		copy(tmp, s.Heights)
		for i := cx; i < cx+tile.W; i++ {
			tmp[i] = cy + tile.H
		}

		u := unevenness(tmp)

		//fmt.Printf("x: %d \tholes: %d \tu: %d\n", cx, holes, u)

		cost := holes*WeightHoles + u*WeightUneven

		if cost < bestCost {
			bestCost = cost
			bestX = cx
			bestY = cy
		}
	}

	// commit
	for i := bestX; i < bestX+tile.W; i++ {
		s.Heights[i] = bestY + tile.H
	}
	// printOut(s.Heights)
	// gap healing
	s.Heights = healNarrowGaps(s.Heights, minRemainingWidth)
	/*
		fmt.Println(". . . . . . . . . . . . . . . . . .")
		printOut(s.Heights)
	*/
	tile.X = bestX
	tile.Y = bestY
	return s
}

func calculateMinWidth(tiles []*gridData.Rect, width int) []int {
	ret := make([]int, len(tiles))
	min := width
	for i := len(tiles) - 1; i >= 0; i-- {
		if min > tiles[i].W {
			min = tiles[i].W
		}
		ret[i] = min
	}
	ret = append(ret, 1)
	return ret
}

func PlaceTilesSkyline(images []*gridData.GridImage, gridW int) {
	rects := make([]*gridData.Rect, len(images))
	for i, img := range images {
		rects[i] = img.GetRec(gridW)
	}
	mins := calculateMinWidth(rects, gridW)

	sky := NewSkyline(gridW)

	for i, rect := range rects {
		sky = Place(sky, rect, mins[i+1])
	}

}
