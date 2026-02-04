package dynamic

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

// PropPathToPath converts a PropPath (array of strings) to a Terraform path.Path.
// PropPath originates from Zod validation errors where each element represents a step
// in the path to a property. Numeric strings are treated as list indexes, while
// non-numeric strings are treated as map/object keys.
//
// Examples:
//   - ["foo"] → path.Root("foo")
//   - ["foo", "1"] → path.Root("foo").AtListIndex(1)
//   - ["foo", "1", "bar"] → path.Root("foo").AtListIndex(1).AtMapKey("bar")
//   - ["foo", "bar", "baz"] → path.Root("foo").AtMapKey("bar").AtMapKey("baz")
//
// Parameters:
//   - propPath: An array of strings representing the path to a property
//
// Returns a path.Path representing the same path in Terraform's path system.
// Returns an empty path.Path (path.Empty()) if propPath is nil or empty.
func PropPathToPath(propPath *[]string) path.Path {
	if propPath == nil || len(*propPath) == 0 {
		return path.Empty()
	}

	segments := *propPath

	// Start with the root element
	p := path.Root(segments[0])

	// Process remaining segments
	for i := 1; i < len(segments); i++ {
		segment := segments[i]

		// Try to parse as an integer for list index
		if idx, err := strconv.Atoi(segment); err == nil {
			p = p.AtListIndex(idx)
		} else {
			// Otherwise treat as a map key
			p = p.AtMapKey(segment)
		}
	}

	return p
}
