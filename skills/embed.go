package skillassets

import "embed"

// Catalog contains the built-in portable ShipProof skill packages.
//
//go:embed */SKILL.md */references/* evals/*.json
var Catalog embed.FS
