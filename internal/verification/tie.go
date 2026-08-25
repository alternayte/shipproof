package verification

import (
	"fmt"
	"sort"

	"github.com/alternayte/shipproof/internal/requirements"
)

// BlockerUnplanned names a requirement that the verification plan does not
// cover. The change would ship with an unproven requirement.
const BlockerUnplanned = "unplanned-requirement"

// BlockerUntied names a plan entry that no requirement in the set claims. The
// plan would prove something the source document never asked for.
const BlockerUntied = "untied-plan-entry"

// TieBlocker reports one mismatch between the requirement set and the
// verification plan.
type TieBlocker struct {
	Kind          string `json:"kind"`
	RequirementID string `json:"requirement_id"`
	Detail        string `json:"detail"`
}

// TieCheck compares the requirement set against the requirement identifiers in
// the plan. A mismatch in either direction is a blocker. This is the
// specification-to-code drift check of Section 8.
//
// An invariant takes no part. The spine ties the requirement set to the
// requirement list only.
func TieCheck(set requirements.Set, plan Plan) []TieBlocker {
	planned := map[string]struct{}{}
	for _, item := range plan.Requirements {
		planned[item.ID] = struct{}{}
	}
	adopted := map[string]struct{}{}
	for _, requirement := range set.Requirements {
		adopted[requirement.ID] = struct{}{}
	}

	blockers := []TieBlocker{}
	for _, requirement := range set.Requirements {
		if _, ok := planned[requirement.ID]; !ok {
			blockers = append(blockers, TieBlocker{
				Kind:          BlockerUnplanned,
				RequirementID: requirement.ID,
				Detail:        fmt.Sprintf("requirement %s has no entry in the verification plan", requirement.ID),
			})
		}
	}
	for _, item := range plan.Requirements {
		if _, ok := adopted[item.ID]; !ok {
			blockers = append(blockers, TieBlocker{
				Kind:          BlockerUntied,
				RequirementID: item.ID,
				Detail:        fmt.Sprintf("plan entry %s names no requirement in the requirement set", item.ID),
			})
		}
	}

	sort.SliceStable(blockers, func(i, j int) bool {
		if blockers[i].Kind != blockers[j].Kind {
			return blockers[i].Kind < blockers[j].Kind
		}
		return blockers[i].RequirementID < blockers[j].RequirementID
	})
	return blockers
}
