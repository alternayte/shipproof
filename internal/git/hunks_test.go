package git

import "testing"

func TestParseHunksReadsRangesAndSymbols(t *testing.T) {
	diff := `diff --git a/internal/git/collector.go b/internal/git/collector.go
index 1111111..2222222 100644
--- a/internal/git/collector.go
+++ b/internal/git/collector.go
@@ -190,0 +190,3 @@ func withRetry(operation func() error) error
+	a
+	b
+	c
@@ -240,2 +243,0 @@ func HeadRevision(dir string) (string, error)
-	old
-	old
diff --git a/docs/workflow.md b/docs/workflow.md
--- a/docs/workflow.md
+++ b/docs/workflow.md
@@ -10 +10 @@
-old
+new
`

	files := ParseHunks(diff)
	if len(files) != 2 {
		t.Fatalf("files = %d, want 2", len(files))
	}
	if files[0].Path != "internal/git/collector.go" {
		t.Errorf("path = %q", files[0].Path)
	}
	if len(files[0].Hunks) != 2 {
		t.Fatalf("hunks = %d, want 2", len(files[0].Hunks))
	}
	first := files[0].Hunks[0]
	if first.StartLine != 190 || first.LineCount != 3 {
		t.Errorf("first hunk = %d,%d, want 190,3", first.StartLine, first.LineCount)
	}
	if first.Symbol != "func withRetry(operation func() error) error" {
		t.Errorf("symbol = %q", first.Symbol)
	}
	second := files[0].Hunks[1]
	if second.LineCount != 0 {
		t.Errorf("deletion hunk line count = %d, want 0", second.LineCount)
	}
	if files[1].Path != "docs/workflow.md" {
		t.Errorf("path = %q", files[1].Path)
	}
	if files[1].Hunks[0].StartLine != 10 || files[1].Hunks[0].LineCount != 1 {
		t.Errorf("single-line hunk = %d,%d, want 10,1", files[1].Hunks[0].StartLine, files[1].Hunks[0].LineCount)
	}
	if files[1].Hunks[0].Symbol != "" {
		t.Errorf("symbol = %q, want empty", files[1].Hunks[0].Symbol)
	}
}

func TestParseHunksSkipsADeletedFile(t *testing.T) {
	diff := `diff --git a/gone.go b/gone.go
--- a/gone.go
+++ /dev/null
@@ -1,3 +0,0 @@
-a
-b
-c
`
	files := ParseHunks(diff)
	if len(files) != 0 {
		t.Fatalf("files = %d, want 0", len(files))
	}
}

func TestCollectChangedHunksOnARealRepository(t *testing.T) {
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "a.go", "package main\n\nfunc main() {\n}\n")
	writeAndCommit(t, dir, "a.go", "package main\n\nfunc main() {\n\tprintln(1)\n}\n")

	files, err := CollectChangedHunks(dir, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatalf("CollectChangedHunks: %v", err)
	}
	if len(files) != 1 || files[0].Path != "a.go" {
		t.Fatalf("files = %#v", files)
	}
	if files[0].Hunks[0].StartLine != 4 || files[0].Hunks[0].LineCount != 1 {
		t.Errorf("hunk = %#v, want start 4 count 1", files[0].Hunks[0])
	}
}
