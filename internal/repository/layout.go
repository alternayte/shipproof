package repository

import "path/filepath"

var directories = []string{
	".shipproof",
	".shipproof/skills",
	".shipproof/shaping",
	".shipproof/documents",
	".shipproof/decisions",
	".shipproof/templates",
	".shipproof/plans",
	".shipproof/changes",
	".shipproof/runs",
	".shipproof/evidence",
	".shipproof/policies",
}

func directoryPaths(root string) []string {
	paths := make([]string, 0, len(directories))
	for _, directory := range directories {
		paths = append(paths, filepath.Join(root, filepath.FromSlash(directory)))
	}
	return paths
}
