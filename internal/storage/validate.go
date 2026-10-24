package storage

import (
	"fmt"
	"regexp"
)

// idPattern accepts only server-generated identifier shapes: UUIDs
// (8-4-4-4-12 hex). Anything else — including "..", "/" or absolute
// paths — is rejected before it can reach filepath.Join.
var idPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateID returns an error when id is not a well-formed identifier.
// It is the single containment boundary for all filesystem paths built
// from list/task IDs (see CodeQL go/path-injection, scan/tasks.sarif).
func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid identifier: %q", id)
	}
	return nil
}
