package git

import "errors"

var (
	ErrGitNotInstalled = errors.New("git is not installed or not on PATH")
	ErrNotGitRepo      = errors.New("not a git repository")
	ErrBadRevision     = errors.New("revision does not exist")
)
