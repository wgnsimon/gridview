package excel

import (
	"fmt"
	"strconv"
	"strings"
)

// PickSheet finds a sheet by name or 1-based index; empty picks the first.
func PickSheet(names []string, want string) (int, error) {
	if len(names) == 0 {
		return 0, fmt.Errorf("workbook has no sheets")
	}
	if want == "" {
		return 0, nil
	}
	for i, n := range names {
		if n == want {
			return i, nil
		}
	}
	for i, n := range names {
		if strings.EqualFold(n, want) {
			return i, nil
		}
	}
	if n, err := strconv.Atoi(want); err == nil && n >= 1 && n <= len(names) {
		return n - 1, nil
	}
	return 0, fmt.Errorf("sheet %q not found, sheets: %s", want, strings.Join(names, ", "))
}
