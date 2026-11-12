// release/version/version.go

// Package version compares release tags with optional numeric parts and text suffixes.
package version

import (
	"regexp"
	"strconv"
	"strings"
)

var pattern = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:[-+](.*))?$`)

// Compare orders release tags; unparseable input compares equal.
func Compare(a, b string) int {
	left := pattern.FindStringSubmatch(strings.TrimSpace(a))
	right := pattern.FindStringSubmatch(strings.TrimSpace(b))
	if left == nil || right == nil {
		return 0
	}
	for i := 1; i <= 3; i++ {
		l, _ := strconv.ParseFloat(left[i], 64)
		r, _ := strconv.ParseFloat(right[i], 64)
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	}
	if left[4] == right[4] {
		return 0
	}
	if left[4] == "" {
		return 1
	}
	if right[4] == "" {
		return -1
	}
	return strings.Compare(left[4], right[4])
}

// Trim drops one leading v from a release tag.
func Trim(v string) string { return strings.TrimPrefix(v, "v") }
