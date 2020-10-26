package inference

import "path/filepath"

// dirWithoutDot returns the directory name of the given path. Unlike filepath.Dir,
// this function will return an empty string (instead of a `.`) to indicate an empty
// directory name.
func dirWithoutDot(path string) string {
	if dir := filepath.Dir(path); dir != "." {
		return dir
	}
	return ""
}

// ancestorDirs returns all ancestor dirnames of the given path. The last element of
// the returned slice will always be empty (indicating the repository root).
func ancestorDirs(path string) (ancestors []string) {
	dir := dirWithoutDot(path)
	for dir != "" {
		ancestors = append(ancestors, dir)
		dir = dirWithoutDot(dir)
	}

	ancestors = append(ancestors, "")
	return ancestors
}

// containsSegment returns true if the given path contains the given segment.
func containsSegment(path, segment string) bool {
	if path == "" {
		return segment == ""
	}

	dir, file := filepath.Split(filepath.Clean(path))
	if file == segment {
		return true
	}

	return containsSegment(dir, segment)
}
