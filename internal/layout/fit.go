package layout

const minWidth = 6

// Fit shrinks cols so that n columns plus separators fit into limit and returns n.
// Wide columns shrink first, narrow ones keep their width where possible.
func Fit(cols []Column, limit, sep int) int {
	for n := len(cols); n > 1; n-- {
		if fit(cols[:n], limit-sep*(n-1)) {
			return n
		}
	}
	if len(cols) == 0 {
		return 0
	}
	cols[0].Width = max(1, min(cols[0].Width, limit))
	return 1
}

func fit(cols []Column, budget int) bool {
	total, floor, widest := 0, 0, 0
	for _, c := range cols {
		total += c.Width
		floor += min(c.Width, minWidth)
		widest = max(widest, c.Width)
	}
	if total <= budget {
		return true
	}
	if floor > budget {
		return false
	}
	lo, hi := minWidth, widest
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if capped(cols, mid) <= budget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	spare := budget - capped(cols, lo)
	for i := range cols {
		if cols[i].Width > lo {
			cols[i].Width = lo
			if spare > 0 {
				cols[i].Width++
				spare--
			}
		}
	}
	return true
}

func capped(cols []Column, limit int) int {
	sum := 0
	for _, c := range cols {
		sum += min(c.Width, limit)
	}
	return sum
}
