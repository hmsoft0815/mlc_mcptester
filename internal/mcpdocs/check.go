package mcpdocs

import "strings"

// FirstDifference compares a written reference with a freshly rendered one
// and returns the first line that differs, or 0 if they match. Line endings
// are ignored, so a checkout with CRLF still matches.
func FirstDifference(written, rendered string) int {
	have := strings.Split(strings.ReplaceAll(written, "\r\n", "\n"), "\n")
	want := strings.Split(rendered, "\n")
	for i := range max(len(have), len(want)) {
		if i >= len(have) || i >= len(want) || have[i] != want[i] {
			return i + 1
		}
	}
	return 0
}
