package reader

const sniffLines = 20

// candidates in order of preference when several fit equally well.
var candidates = [...]byte{'\t', ';', ',', '|'}

// sniffDelimiter guesses the delimiter from the first lines of a CSV file.
// A delimiter that appears equally often on every line wins.
func sniffDelimiter(sample []byte, complete bool) rune {
	lines := countPerLine(sample, complete)
	best, bestLines := byte(','), 0
	for _, c := range candidates {
		if consistent(lines, c) {
			return rune(c)
		}
		n := 0
		for _, l := range lines {
			if l[c] > 0 {
				n++
			}
		}
		if n > bestLines {
			best, bestLines = c, n
		}
	}
	return rune(best)
}

func consistent(lines []map[byte]int, c byte) bool {
	if len(lines) == 0 || lines[0][c] == 0 {
		return false
	}
	for _, l := range lines[1:] {
		if l[c] != lines[0][c] {
			return false
		}
	}
	return true
}

func countPerLine(sample []byte, complete bool) []map[byte]int {
	var lines []map[byte]int
	cur := map[byte]int{}
	quoted, empty := false, true
	for _, b := range sample {
		switch {
		case b == '"':
			quoted = !quoted
		case quoted:
		case b == '\n':
			if !empty {
				lines = append(lines, cur)
				if len(lines) == sniffLines {
					return lines
				}
			}
			cur, empty = map[byte]int{}, true
			continue
		case b == '\t' || b == ';' || b == ',' || b == '|':
			cur[b]++
		}
		if b != '\r' {
			empty = false
		}
	}
	if complete && !empty {
		lines = append(lines, cur)
	}
	return lines
}
