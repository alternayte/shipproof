package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/alternayte/shipproof/internal/change"
)

func runChange(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof change <start|status|check> ...")
		return 2
	}

	switch args[0] {
	case "start":
		return runChangeStart(args[1:], stdout, stderr)
	case "status":
		return runChangeStatus(args[1:], stdout, stderr)
	case "check":
		return runChangeCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown change command %q\n", args[0])
		return 2
	}
}

func runChangeStart(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: shipproof change start <change-id> --source <path> [--shaping <session-id>] [--ceremony 0|1|2|3] [--force]")
		return 2
	}

	changeID := args[0]
	var source, shapingRef string
	var ceremony *int
	force := false
	for index := 1; index < len(args); index++ {
		switch args[index] {
		case "--source":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--source requires a path")
				return 2
			}
			source = args[index+1]
			index++
		case "--shaping":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--shaping requires a session id")
				return 2
			}
			shapingRef = args[index+1]
			index++
		case "--ceremony":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--ceremony requires a level from 0 to 3")
				return 2
			}
			parsed, err := strconv.Atoi(args[index+1])
			if err != nil {
				fmt.Fprintf(stderr, "--ceremony requires a level from 0 to 3; got %q\n", args[index+1])
				return 2
			}
			if parsed < 0 || parsed > change.MaxCeremony {
				fmt.Fprintf(stderr, "--ceremony requires a level from 0 to %d; got %q\n", change.MaxCeremony, args[index+1])
				return 2
			}
			ceremony = &parsed
			index++
		case "--force":
			force = true
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[index])
			return 2
		}
	}

	if source == "" {
		fmt.Fprintln(stderr, "--source is required")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	var record change.Record
	if force {
		record, err = change.Restart(root, changeID, source, shapingRef, ceremony)
	} else {
		level := change.DefaultCeremony
		if ceremony != nil {
			level = *ceremony
		}
		record, err = change.Start(root, changeID, source, shapingRef, level)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	rel, _ := filepath.Rel(root, change.Path(root, changeID))
	fmt.Fprintf(stdout, "Started change %s\n", changeID)
	fmt.Fprintf(stdout, "Source: %s\n", record.SourcePath)
	fmt.Fprintf(stdout, "Snapshot: %s\n", record.SnapshotPath)
	fmt.Fprintf(stdout, "SHA-256: %s\n", record.SHA256)
	if record.ShapingRef != "" {
		fmt.Fprintf(stdout, "Shaping: %s\n", record.ShapingRef)
	}
	fmt.Fprintf(stdout, "Ceremony: %d\n", record.CeremonyLevel())
	fmt.Fprintf(stdout, "Captured: %s\n", record.CapturedAt)
	fmt.Fprintf(stdout, "Record: %s\n", filepath.ToSlash(rel))
	return 0
}

func runChangeStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: shipproof change status <change-id>")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	record, err := change.Load(root, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	hasPlan := change.HasVerificationPlan(root, args[0])
	planStatus := "absent"
	if hasPlan {
		planStatus = "present"
	}

	fmt.Fprintf(stdout, "Change: %s\n", record.ChangeID)
	fmt.Fprintf(stdout, "Source: %s\n", record.SourcePath)
	fmt.Fprintf(stdout, "Snapshot: %s\n", record.SnapshotPath)
	fmt.Fprintf(stdout, "SHA-256: %s\n", record.SHA256)
	if record.ShapingRef != "" {
		fmt.Fprintf(stdout, "Shaping: %s\n", record.ShapingRef)
	}
	fmt.Fprintf(stdout, "Captured: %s\n", record.CapturedAt)
	fmt.Fprintf(stdout, "Verification plan: %s\n", planStatus)

	stale, err := record.Staleness(root)
	if err != nil {
		fmt.Fprintf(stdout, "Intent source: unknown (%v)\n", err)
		return 0
	}
	if stale.Stale {
		fmt.Fprintf(stdout, "Intent source: stale (snapshot %s, current %s)\n", stale.SnapshotHash, stale.CurrentHash)
	} else {
		fmt.Fprintln(stdout, "Intent source: current")
	}
	return 0
}

func runChangeCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof change check <change-id>")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	record, err := change.Load(root, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := record.VerifyHash(root); err != nil {
		fmt.Fprintf(stderr, "hash verification failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Change %s is valid: snapshot hash matches.\n", record.ChangeID)

	stale, err := record.Staleness(root)
	if err != nil {
		fmt.Fprintf(stdout, "Intent source: unknown (%v)\n", err)
		return 0
	}
	if stale.Stale {
		fmt.Fprintf(stdout, "Intent source: stale. Re-verify this change before using its evidence.\n")
	} else {
		fmt.Fprintln(stdout, "Intent source: current.")
	}
	return 0
}
