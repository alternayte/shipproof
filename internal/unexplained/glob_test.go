package unexplained

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"docs/**", "docs/workflow.md", true},
		{"docs/**", "docs/design/a.md", true},
		{"docs/**", "docs", false},
		{"docs/**", "internal/docs/a.md", false},
		{"CHANGELOG.md", "CHANGELOG.md", true},
		{"CHANGELOG.md", "docs/CHANGELOG.md", false},
		{"*.md", "README.md", true},
		{"*.md", "docs/README.md", false},
		{"**/*.md", "docs/design/a.md", true},
		{"**/*.md", "README.md", true},
		{"internal/*/app.go", "internal/cli/app.go", true},
		{"internal/*/app.go", "internal/a/b/app.go", false},
	}
	for _, testCase := range cases {
		if got := Match(testCase.pattern, testCase.path); got != testCase.want {
			t.Errorf("Match(%q, %q) = %v, want %v", testCase.pattern, testCase.path, got, testCase.want)
		}
	}
}
