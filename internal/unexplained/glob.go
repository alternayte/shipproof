package unexplained

import (
	"path"
	"strings"
)

// Match reports whether one path matches one glob pattern.
//
// The syntax is the small subset the configuration needs. A single star
// matches inside one path segment. A double star matches zero or more whole
// segments. No dependency exists for this, and the module has none.
func Match(pattern, target string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(target, "/"))
}

func matchSegments(pattern, target []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			if len(pattern) == 1 {
				// A trailing ** needs at least one remaining segment. The
				// directory itself is not a file the report can name.
				return len(target) > 0
			}
			for index := 0; index <= len(target); index++ {
				if matchSegments(pattern[1:], target[index:]) {
					return true
				}
			}
			return false
		}
		if len(target) == 0 {
			return false
		}
		if ok, err := path.Match(pattern[0], target[0]); err != nil || !ok {
			return false
		}
		pattern, target = pattern[1:], target[1:]
	}
	return len(target) == 0
}
