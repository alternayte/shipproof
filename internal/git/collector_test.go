package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectCommits(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "file1.go", "package main")
	writeAndCommit(t, dir, "file2.go", "package util")

	commits, err := CollectCommits(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CollectCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	for _, c := range commits {
		if c.Hash == "" {
			t.Error("commit hash is empty")
		}
		if c.Author == "" {
			t.Error("commit author is empty")
		}
		if c.Timestamp == "" {
			t.Error("commit timestamp is empty")
		}
		if c.Provenance != "observed" {
			t.Errorf("commit %s: provenance = %s, want observed", c.Hash[:7], c.Provenance)
		}
	}
}

func TestCollectCommitsEmptyRange(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)

	commits, err := CollectCommits(dir, "master", "master")
	if err != nil {
		t.Fatalf("CollectCommits: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits, got %d", len(commits))
	}
}

func TestCollectChangedFiles(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "src/a.go", "package a")
	writeAndCommit(t, dir, "src/b.go", "package b")

	files, err := CollectChangedFiles(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CollectChangedFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	foundA, foundB := false, false
	for _, f := range files {
		if f == "src/a.go" {
			foundA = true
		}
		if f == "src/b.go" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("missing expected files in %v", files)
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}")

	adds, dels, err := CountLines(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CountLines: %v", err)
	}
	if adds != 5 {
		t.Errorf("additions = %d, want 5", adds)
	}
	if dels != 0 {
		t.Errorf("deletions = %d, want 0", dels)
	}

	writeAndCommit(t, dir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}")

	adds, dels, err = CountLines(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CountLines: %v", err)
	}
	if adds != 5 {
		t.Errorf("additions = %d, want 5", adds)
	}
	if dels != 0 {
		t.Errorf("deletions = %d, want 0", dels)
	}
}

func TestDiffStat(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "app.go", "package app\n\nfunc Run() {}")

	stat, err := DiffStat(dir, "initial", "master")
	if err != nil {
		t.Fatalf("DiffStat: %v", err)
	}
	if !strings.Contains(stat, "insertion") && !strings.Contains(stat, "file changed") {
		t.Errorf("DiffStat missing expected content: %s", stat)
	}
}

func TestCollectMetadata(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "x.go", "package x")

	md, err := CollectMetadata(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CollectMetadata: %v", err)
	}
	if len(md.Commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(md.Commits))
	}
	if len(md.ChangedFiles) != 1 {
		t.Errorf("expected 1 changed file, got %d", len(md.ChangedFiles))
	}
	if md.ChangedFiles[0] != "x.go" {
		t.Errorf("changed file = %q, want x.go", md.ChangedFiles[0])
	}
	if md.Additions < 1 {
		t.Errorf("additions = %d, want at least 1", md.Additions)
	}
	if md.Provenance != "observed" {
		t.Errorf("metadata provenance = %s, want observed", md.Provenance)
	}
}

func TestMetadataProvenance(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "a.go", "package a")
	writeAndCommit(t, dir, "b.go", "package b")

	md, err := CollectMetadata(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CollectMetadata: %v", err)
	}

	if md.Provenance != "observed" {
		t.Errorf("metadata provenance = %s, want observed", md.Provenance)
	}
	for _, c := range md.Commits {
		if c.Provenance != "observed" {
			t.Errorf("commit %s provenance = %s, want observed", c.Hash[:7], c.Provenance)
		}
	}
}

func TestBadRevision(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)

	_, err := CollectCommits(dir, "nonexistent", "master")
	if err == nil {
		t.Fatal("expected error for bad revision")
	}
	if !errors.Is(err, ErrBadRevision) {
		t.Errorf("error = %v, want ErrBadRevision", err)
	}
}

func TestNotGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	_, err := CollectCommits(dir, "main", "main")
	if err == nil {
		t.Fatal("expected error for non-repo dir")
	}
	if !errors.Is(err, ErrNotGitRepo) {
		t.Errorf("error = %v, want ErrNotGitRepo", err)
	}
}

func TestInvariantProvenance(t *testing.T) {
	t.Parallel()
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "f.go", "package f")

	md, err := CollectMetadata(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CollectMetadata: %v", err)
	}

	if md.Provenance != "observed" {
		t.Error("metadata provenance is not observed")
	}
	for i, c := range md.Commits {
		if c.Provenance != "observed" {
			t.Errorf("commit[%d] provenance is not observed", i)
		}
	}

	commits, err := CollectCommits(dir, "initial", "master")
	if err != nil {
		t.Fatalf("CollectCommits: %v", err)
	}
	for i, c := range commits {
		if c.Provenance != "observed" {
			t.Errorf("CollectCommits commit[%d] provenance is not observed", i)
		}
	}
}

func TestInvariantErrorOnFail(t *testing.T) {
	t.Parallel()

	md, err := CollectMetadata(t.TempDir(), "main", "main")
	if err == nil {
		t.Fatal("expected error for non-repo dir")
	}
	if md.Commits != nil || md.ChangedFiles != nil {
		t.Error("expected nil slices on error")
	}
	if md.Additions != 0 || md.Deletions != 0 {
		t.Error("expected zero counts on error")
	}

	adds, dels, err := CountLines(t.TempDir(), "main", "main")
	if err == nil {
		t.Fatal("expected error for non-repo dir during CountLines")
	}
	if adds != 0 || dels != 0 {
		t.Error("expected zero counts on error")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit(t, dir, "init", "--initial-branch=master")
	runGit(t, dir, "config", "user.email", "test@shipproof.test")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "commit", "--allow-empty", "-m", "initial commit")
	runGit(t, dir, "tag", "initial")

	return dir
}

func writeAndCommit(t *testing.T, dir, path, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", path)
	runGit(t, dir, "commit", "-m", "add "+path)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func TestResolveGitHubRepoHTTPS(t *testing.T) {
	t.Parallel()

	dir := initTestRepo(t)
	runGit(t, dir, "remote", "add", "origin", "https://github.com/acme/widget.git")

	owner, name, err := ResolveGitHubRepo(dir)
	if err != nil {
		t.Fatalf("ResolveGitHubRepo: %v", err)
	}
	if owner != "acme" || name != "widget" {
		t.Errorf("owner/name = %q/%q, want acme/widget", owner, name)
	}
}

func TestResolveGitHubRepoSSH(t *testing.T) {
	t.Parallel()

	dir := initTestRepo(t)
	runGit(t, dir, "remote", "add", "origin", "git@github.com:acme/widget.git")

	owner, name, err := ResolveGitHubRepo(dir)
	if err != nil {
		t.Fatalf("ResolveGitHubRepo: %v", err)
	}
	if owner != "acme" || name != "widget" {
		t.Errorf("owner/name = %q/%q, want acme/widget", owner, name)
	}
}

func TestResolveGitHubRepoNoRemote(t *testing.T) {
	t.Parallel()

	dir := initTestRepo(t)

	_, _, err := ResolveGitHubRepo(dir)
	if err != ErrNoRemote {
		t.Fatalf("expected ErrNoRemote, got %v", err)
	}
}

func TestResolveGitHubRepoNonGitHubRemote(t *testing.T) {
	t.Parallel()

	dir := initTestRepo(t)
	runGit(t, dir, "remote", "add", "origin", "https://gitlab.com/acme/widget.git")

	_, _, err := ResolveGitHubRepo(dir)
	if err == nil {
		t.Fatal("expected error for non-GitHub remote")
	}
	if !strings.Contains(err.Error(), "not a GitHub remote") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseGitHubURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		url         string
		owner, name string
		wantOK      bool
	}{
		{"https://github.com/acme/widget.git", "acme", "widget", true},
		{"https://github.com/acme/widget", "acme", "widget", true},
		{"http://github.com/acme/widget.git", "acme", "widget", true},
		{"git@github.com:acme/widget.git", "acme", "widget", true},
		{"git@github.com:acme/widget", "acme", "widget", true},
		{"https://gitlab.com/acme/widget.git", "", "", false},
		{"https://github.com/onlyowner", "", "", false},
		{"not-a-url", "", "", false},
	}

	for _, tc := range cases {
		owner, name, ok := parseGitHubURL(tc.url)
		if ok != tc.wantOK {
			t.Errorf("parseGitHubURL(%q) ok = %v, want %v", tc.url, ok, tc.wantOK)
			continue
		}
		if ok && (owner != tc.owner || name != tc.name) {
			t.Errorf("parseGitHubURL(%q) = %q/%q, want %q/%q", tc.url, owner, name, tc.owner, tc.name)
		}
	}
}

func TestDirtyOutsideIgnoresExcludedPath(t *testing.T) {
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "source.go", "package main\n")

	if err := os.MkdirAll(filepath.Join(dir, ".shipproof", "changes", "SP-001"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".shipproof", "changes", "SP-001", "evidence-pack.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dirty, err := Dirty(dir)
	if err != nil {
		t.Fatalf("Dirty() error = %v", err)
	}
	if !dirty {
		t.Fatal("Dirty() = false; the untracked pack must count as a difference")
	}

	outside, err := DirtyOutside(dir, ".shipproof")
	if err != nil {
		t.Fatalf("DirtyOutside() error = %v", err)
	}
	if outside {
		t.Fatal("DirtyOutside() = true; a write under .shipproof must not count")
	}
}

func TestDirtyOutsideCountsSourceChange(t *testing.T) {
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "source.go", "package main\n")

	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("package main // edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outside, err := DirtyOutside(dir, ".shipproof")
	if err != nil {
		t.Fatalf("DirtyOutside() error = %v", err)
	}
	if !outside {
		t.Fatal("DirtyOutside() = false; a source edit must count")
	}
}

func TestDirtyOutsideCountsUntrackedSource(t *testing.T) {
	dir := initTestRepo(t)
	writeAndCommit(t, dir, "source.go", "package main\n")

	if err := os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outside, err := DirtyOutside(dir, ".shipproof")
	if err != nil {
		t.Fatalf("DirtyOutside() error = %v", err)
	}
	if !outside {
		t.Fatal("DirtyOutside() = false; untracked source must count")
	}
}
