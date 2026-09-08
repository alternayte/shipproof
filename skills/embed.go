package skillassets

import "embed"

// Catalog holds the three instruction files of Section 8.2. They replace the
// old skill catalog. Three files are small enough that an agent reads all of
// them, and a catalog was not.
//
//go:embed capture-intent.md plan-proof.md read-evidence.md
var Catalog embed.FS
