package skillassets

import "embed"

// Catalog holds the three instruction files of Section 8.2. They replace the
// old skill catalog. Three files are small enough that an agent reads all of
// them, and a catalog was not.
//
//go:embed */SKILL.md
var Catalog embed.FS
