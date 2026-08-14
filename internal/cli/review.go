package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/shipproof/shipproof/internal/change"
	"github.com/shipproof/shipproof/internal/review"
)

func runReview(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof review <prepare> ...")
		return 2
	}

	switch args[0] {
	case "prepare":
		return runReviewPrepare(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown review command %q\n", args[0])
		return 2
	}
}

func runReviewPrepare(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof review prepare <change-id>")
		return 2
	}

	changeID := args[0]

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	packet, err := review.Prepare(root, changeID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := review.WritePacket(root, packet); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	packetPath := filepath.Join(root, ".shipproof", "changes", changeID, "review-packet.json")
	rel, _ := filepath.Rel(root, packetPath)
	fmt.Fprintf(stdout, "Review packet written: %s\n", filepath.ToSlash(rel))
	return 0
}
