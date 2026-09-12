package presentation

import "fmt"

// countNoun renders "1 file" / "2 files" style text for user-facing counts.
func countNoun(n int, singular string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %ss", n, singular)
}
