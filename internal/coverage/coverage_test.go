package coverage

import (
	"testing"

	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
)

func requirementSet(ids ...string) requirements.Set {
	set := requirements.Set{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Adopter:       requirements.AdopterNative,
	}
	for _, id := range ids {
		set.Requirements = append(set.Requirements, requirements.Requirement{
			ID: id, Statement: id + " statement", Provenance: requirements.Observed,
		})
	}
	return set
}

func planWith(items ...verification.Item) verification.Plan {
	return verification.Plan{SchemaVersion: "0.1", ChangeID: "SP-021", Requirements: items}
}

func resultSet(results ...proofs.Result) *proofs.Set {
	return &proofs.Set{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Timestamp:     "2026-08-25T10:00:00Z",
		Results:       results,
	}
}

func TestBuildProvenWhenEveryAutomatedProofPassed(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
			{Type: "test", Target: "b", Command: "go test ./b/"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", Status: proofs.Pass},
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 1, Command: "go test ./b/", Status: proofs.Pass},
		),
		true,
	)
	assertRow(t, matrix, 0, Proven, Observed)
}

func TestBuildFailedWhenAnyAutomatedProofFailed(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
			{Type: "test", Target: "b", Command: "go test ./b/"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", Status: proofs.Pass},
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 1, Command: "go test ./b/", ExitCode: 1, Status: proofs.Fail},
		),
		true,
	)
	assertRow(t, matrix, 0, Failed, Observed)
}

func TestBuildAcceptedWhenEveryProofIsHumanAndAccepted(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "human", Target: "a", Human: true, Rationale: "A person reads it.", AcceptedAt: "2026-08-25T10:00:00Z"},
		}}),
		resultSet(proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Status: proofs.Human}),
		true,
	)
	assertRow(t, matrix, 0, Accepted, Human)
}

func TestBuildAwaitingHumanWithNoRecordedAcceptance(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "human", Target: "a", Human: true, Rationale: "A person reads it."},
		}}),
		resultSet(proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Status: proofs.Human}),
		true,
	)
	assertRow(t, matrix, 0, AwaitingHuman, Unknown)
}

func TestBuildAwaitingHumanWhenOneHumanProofOfTwoIsUnaccepted(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "human", Target: "a", Human: true, Rationale: "r", AcceptedAt: "2026-08-25T10:00:00Z"},
			{Type: "human", Target: "b", Human: true, Rationale: "r"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Status: proofs.Human},
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 1, Status: proofs.Human},
		),
		true,
	)
	assertRow(t, matrix, 0, AwaitingHuman, Unknown)
}

func TestBuildUnprovenWithNoPlanEntry(t *testing.T) {
	t.Parallel()

	matrix := Build(requirementSet("SP-021-R1"), planWith(), nil, false)
	assertRow(t, matrix, 0, Unproven, Unknown)
}

func TestBuildUnprovenWhenNoProofRan(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
		}}),
		nil,
		false,
	)
	assertRow(t, matrix, 0, Unproven, Unknown)
}

func TestBuildUnprovenWhenTheResultsAreNotCurrent(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
		}}),
		resultSet(proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", Status: proofs.Pass}),
		false,
	)
	assertRow(t, matrix, 0, Unproven, Unknown)
	if matrix.RunCurrent {
		t.Fatal("RunCurrent = true, want false")
	}
}

func TestBuildFailedBeatsProvenAcrossAMixedItem(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
			{Type: "human", Target: "b", Human: true, Rationale: "r"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", ExitCode: 2, Status: proofs.Fail},
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 1, Status: proofs.Human},
		),
		true,
	)
	assertRow(t, matrix, 0, Failed, Observed)
}

func TestBuildProvenWhenAnAutomatedProofPassesBesideAnUnacceptedHumanProof(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
			{Type: "human", Target: "b", Human: true, Rationale: "r"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", Status: proofs.Pass},
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 1, Status: proofs.Human},
		),
		true,
	)
	assertRow(t, matrix, 0, Proven, Observed)
}

func TestBuildKeepsTheSidecarOrderAndStatements(t *testing.T) {
	t.Parallel()

	matrix := Build(requirementSet("SP-021-R2", "SP-021-R1"), planWith(), nil, false)
	if len(matrix.Rows) != 2 {
		t.Fatalf("Rows = %d, want 2", len(matrix.Rows))
	}
	if matrix.Rows[0].RequirementID != "SP-021-R2" {
		t.Fatalf("row 1 = %q, want the sidecar order", matrix.Rows[0].RequirementID)
	}
	if matrix.Rows[0].Statement != "SP-021-R2 statement" {
		t.Fatalf("row 1 statement = %q", matrix.Rows[0].Statement)
	}
	if matrix.ChangeID != "SP-021" {
		t.Fatalf("ChangeID = %q", matrix.ChangeID)
	}
}

func TestBuildNeverReportsInferred(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		matrix Matrix
	}{
		{"proven", Build(requirementSet("SP-021-R1"), planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{{Type: "test", Target: "a", Command: "x"}}}), resultSet(proofs.Result{RequirementID: "SP-021-R1", Command: "x", Status: proofs.Pass}), true)},
		{"unproven", Build(requirementSet("SP-021-R1"), planWith(), nil, false)},
		{"awaiting", Build(requirementSet("SP-021-R1"), planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{{Type: "human", Target: "a", Human: true, Rationale: "r"}}}), resultSet(proofs.Result{RequirementID: "SP-021-R1", Status: proofs.Human}), true)},
	}
	for _, testCase := range cases {
		for _, row := range testCase.matrix.Rows {
			if string(row.Provenance) == "inferred" {
				t.Fatalf("%s: row %s reads inferred provenance", testCase.name, row.RequirementID)
			}
			switch row.Provenance {
			case Observed, Human, Unknown:
			default:
				t.Fatalf("%s: row %s reads provenance %q", testCase.name, row.RequirementID, row.Provenance)
			}
		}
	}
}

func assertRow(t *testing.T, matrix Matrix, index int, state State, provenance Provenance) {
	t.Helper()

	if len(matrix.Rows) <= index {
		t.Fatalf("Rows = %d, want more than %d", len(matrix.Rows), index)
	}
	row := matrix.Rows[index]
	if row.State != state {
		t.Fatalf("row %d state = %q, want %q", index, row.State, state)
	}
	if row.Provenance != provenance {
		t.Fatalf("row %d provenance = %q, want %q", index, row.Provenance, provenance)
	}
}

func TestBuildUnprovenWhenARecordedPassNamesADifferentCommand(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "b", Command: "go test ./b/"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", Status: proofs.Pass},
		),
		true,
	)
	assertRow(t, matrix, 0, Unproven, Unknown)
}

func TestBuildUnprovenWhenARecordedFailureNamesADifferentCommand(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "b", Command: "go test ./b/"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", ExitCode: 1, Status: proofs.Fail},
		),
		true,
	)
	assertRow(t, matrix, 0, Unproven, Unknown)
}

func TestBuildUnprovenWhenOnlyOneOfTwoCommandsStillMatches(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "test", Target: "a", Command: "go test ./a/"},
			{Type: "test", Target: "b", Command: "go test ./b/"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./a/", Status: proofs.Pass},
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 1, Command: "go test ./old/", Status: proofs.Pass},
		),
		true,
	)
	assertRow(t, matrix, 0, Unproven, Unknown)
}

func TestBuildAcceptedWhenAHumanProofCarriesAnEmptyCommand(t *testing.T) {
	t.Parallel()

	matrix := Build(
		requirementSet("SP-021-R1"),
		planWith(verification.Item{ID: "SP-021-R1", Proof: []verification.Proof{
			{Type: "human", Target: "a", Human: true, Rationale: "A person reads it.", AcceptedAt: "2026-08-25T10:00:00Z"},
		}}),
		resultSet(
			proofs.Result{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "", Status: proofs.Human},
		),
		true,
	)
	assertRow(t, matrix, 0, Accepted, Human)
}
