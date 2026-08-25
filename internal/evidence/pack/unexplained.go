package pack

import (
	"os"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/covprofile"
	"github.com/alternayte/shipproof/internal/git"
	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/repository"
	"github.com/alternayte/shipproof/internal/schema"
	"github.com/alternayte/shipproof/internal/unexplained"
	"github.com/alternayte/shipproof/internal/verification"
)

// coverageChecks turns the requirement coverage matrix into one check per
// requirement. A matrix row never reads inferred, so no check does either.
func coverageChecks(matrix coverage.Matrix) []schema.Check {
	checks := make([]schema.Check, 0, len(matrix.Rows))
	for _, row := range matrix.Rows {
		checks = append(checks, schema.Check{
			ID:         "coverage:" + row.RequirementID,
			Status:     checkStatus(row.State),
			Source:     "shipproof-coverage",
			Provenance: checkProvenance(row.Provenance),
		})
	}
	return checks
}

// checkStatus maps a matrix state onto the four pack statuses.
func checkStatus(state coverage.State) string {
	switch state {
	case coverage.Proven, coverage.Accepted:
		return "pass"
	case coverage.Failed:
		return "fail"
	default:
		return "unknown"
	}
}

// checkProvenance maps a matrix provenance onto the pack vocabulary. The pack
// has no unknown provenance. A row that nothing observed is derived, because
// ShipProof computed it from the artifacts on disk.
func checkProvenance(provenance coverage.Provenance) schema.ProvenanceKind {
	switch provenance {
	case coverage.Observed:
		return schema.ProvenanceObserved
	case coverage.Human:
		return schema.ProvenanceHuman
	default:
		return schema.ProvenanceDerived
	}
}

// buildUnexplained derives the unexplained-change section. It returns nil when
// no revision range is available, because no changed line is then known.
func buildUnexplained(root, changeID string, plan verification.Plan, base, head string) *schema.UnexplainedEvidence {
	if base == "" {
		return nil
	}
	files, err := git.CollectChangedHunks(root, base, head)
	if err != nil {
		return nil
	}

	var profile *covprofile.Profile
	if _, err := os.Stat(proofs.MergedProfilePath(root, changeID)); err == nil {
		profile, _ = covprofile.ParseFile(proofs.MergedProfilePath(root, changeID), covprofile.ModulePath(root))
	}

	var ignore []string
	if cfg, err := repository.LoadConfig(root); err == nil {
		ignore = cfg.Verification.UnexplainedIgnore
	}

	report := unexplained.Build(unexplained.Input{
		ChangeID: changeID,
		Files:    files,
		Profile:  profile,
		Targets:  planTargets(plan),
		Ignore:   ignore,
	})

	section := &schema.UnexplainedEvidence{
		CoverageAvailable:   report.CoverageAvailable,
		UninstrumentedLines: report.UninstrumentedLines,
	}
	for _, finding := range report.LineFindings {
		section.LineFindings = append(section.LineFindings, schema.UnexplainedLine{
			File: finding.File, Symbol: finding.Symbol,
			StartLine: finding.StartLine, EndLine: finding.EndLine,
		})
	}
	for _, finding := range report.FileFindings {
		section.FileFindings = append(section.FileFindings, schema.UnexplainedFile{
			Path: finding.Path, IgnorePattern: finding.IgnorePattern,
		})
	}
	return section
}

// planTargets lists every proof target in the plan.
func planTargets(plan verification.Plan) []string {
	var targets []string
	for _, group := range [][]verification.Item{plan.Requirements, plan.Invariants} {
		for _, item := range group {
			for _, proof := range item.Proof {
				if proof.Target != "" {
					targets = append(targets, proof.Target)
				}
			}
		}
	}
	return targets
}
