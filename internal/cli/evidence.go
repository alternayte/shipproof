package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/shipproof/shipproof/internal/change"
	"github.com/shipproof/shipproof/internal/evidence/pack"
	"github.com/shipproof/shipproof/internal/git"
	"github.com/shipproof/shipproof/internal/github"
	"github.com/shipproof/shipproof/internal/schema"
)

var githubAPIURLOverride = ""

func newGitHubClient(token string) (*github.Client, error) {
	if githubAPIURLOverride != "" {
		return github.NewClientWithURL(token, githubAPIURLOverride)
	}
	return github.NewClient(token)
}

func runEvidence(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof evidence <pack|review> ...")
		return 2
	}

	switch args[0] {
	case "pack":
		return runEvidencePack(args[1:], stdout, stderr)
	case "review":
		return runEvidenceReview(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown evidence command %q\n", args[0])
		return 2
	}
}

func runEvidencePack(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: shipproof evidence pack <change-id> [--base <rev>] [--head <rev>] [--evidence <file>...]")
		return 2
	}

	changeID := args[0]
	var opts pack.Options

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--base":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--base requires a revision")
				return 2
			}
			i++
			opts.BaseRev = args[i]
		case "--head":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--head requires a revision")
				return 2
			}
			i++
			opts.HeadRev = args[i]
		case "--evidence":
			for i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				i++
				opts.EvidenceFiles = append(opts.EvidenceFiles, args[i])
			}
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[i])
			return 2
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	assembled, err := pack.Assemble(root, changeID, opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := pack.WritePack(root, assembled); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	packPath := filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json")
	rel, _ := filepath.Rel(root, packPath)
	fmt.Fprintf(stdout, "Evidence pack written: %s\n", filepath.ToSlash(rel))
	return 0
}

func runEvidenceReview(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: shipproof evidence review <change-id>")
		return 2
	}

	changeID := args[0]

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	packPath := filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json")
	data, err := os.ReadFile(packPath)
	if err != nil {
		fmt.Fprintf(stderr, "evidence pack not found for change %q; run shipproof evidence pack %s first\n", changeID, changeID)
		return 1
	}

	var evidencePack schema.EvidencePack
	if err := json.Unmarshal(data, &evidencePack); err != nil {
		fmt.Fprintf(stderr, "parse evidence pack: %v\n", err)
		return 1
	}

	if len(evidencePack.Implementation.Commits) == 0 {
		fmt.Fprintf(stderr, "evidence pack for %q has no commits; run shipproof evidence pack %s with --base and --head\n", changeID, changeID)
		return 1
	}

	token, err := github.ResolveToken()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	owner, name, err := git.ResolveGitHubRepo(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve GitHub repository: %v\n", err)
		return 1
	}

	client, err := newGitHubClient(token)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	var pr *github.PullRequest
	for _, commit := range evidencePack.Implementation.Commits {
		found, lookupErr := client.FindPRByCommit(owner, name, commit.Hash)
		if lookupErr == github.ErrNotFound {
			continue
		}
		if lookupErr != nil {
			fmt.Fprintf(stderr, "GitHub API error: %v\n", lookupErr)
			return 1
		}
		pr = found
		break
	}

	if pr == nil {
		fmt.Fprintf(stderr, "no GitHub pull request found for any commit of %q\n", changeID)
		return 1
	}

	review := buildReviewEvidence(pr)
	if err := writeReviewFile(root, changeID, review); err != nil {
		fmt.Fprintf(stderr, "write review.json: %v\n", err)
		return 1
	}

	rel, _ := filepath.Rel(root, filepath.Join(root, ".shipproof", "changes", changeID, "review.json"))
	fmt.Fprintf(stdout, "Review data written: %s\n", filepath.ToSlash(rel))
	fmt.Fprintf(stdout, "PR: #%d %s\n", review.PRNumber, review.PRURL)
	if review.FirstReviewAt != "" {
		fmt.Fprintf(stdout, "First review: %s\n", review.FirstReviewAt)
	} else {
		fmt.Fprintln(stdout, "First review: none submitted")
	}
	fmt.Fprintf(stdout, "Reviews: %d, comments: %d, reviewers: %d\n", review.ReviewCount, review.CommentCount, review.DistinctReviewers)
	return 0
}

func buildReviewEvidence(pr *github.PullRequest) schema.ReviewEvidence {
	review := schema.ReviewEvidence{
		Source:       "github",
		PRNumber:     pr.Number,
		PRURL:        pr.URL,
		OpenedAt:     pr.CreatedAt,
		ReviewCount:  pr.Reviews.TotalCount,
		CommentCount: pr.Reviews.TotalCount + pr.ReviewThreads.TotalCount,
		State:        pr.State,
		CollectedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	reviewers := make(map[string]struct{})
	var logins []string
	for _, r := range pr.Reviews.Nodes {
		if r.SubmittedAt != "" && (review.FirstReviewAt == "" || r.SubmittedAt < review.FirstReviewAt) {
			review.FirstReviewAt = r.SubmittedAt
		}
		if r.Author.Login != "" {
			if _, seen := reviewers[r.Author.Login]; !seen {
				reviewers[r.Author.Login] = struct{}{}
				logins = append(logins, r.Author.Login)
			}
		}
	}
	review.DistinctReviewers = len(reviewers)
	review.ReviewerLogins = logins

	return review
}

func writeReviewFile(root, changeID string, review schema.ReviewEvidence) error {
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create change directory: %w", err)
	}

	data, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return fmt.Errorf("encode review data: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(filepath.Join(dir, "review.json"), data, 0o644); err != nil {
		return fmt.Errorf("write review.json: %w", err)
	}

	return nil
}
