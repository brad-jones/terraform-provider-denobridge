package dynamic

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestPropPathToPath(t *testing.T) {
	tests := []struct {
		name      string
		propPath  *[]string
		expected  path.Path
		expectNil bool
	}{
		{
			name:      "nil propPath",
			propPath:  nil,
			expected:  path.Empty(),
			expectNil: true,
		},
		{
			name:      "empty propPath",
			propPath:  &[]string{},
			expected:  path.Empty(),
			expectNil: true,
		},
		{
			name:     "single root element",
			propPath: &[]string{"foo"},
			expected: path.Root("foo"),
		},
		{
			name:     "root with list index",
			propPath: &[]string{"foo", "1"},
			expected: path.Root("foo").AtListIndex(1),
		},
		{
			name:     "root with list index and map key",
			propPath: &[]string{"foo", "1", "bar"},
			expected: path.Root("foo").AtListIndex(1).AtMapKey("bar"),
		},
		{
			name:     "root with multiple map keys",
			propPath: &[]string{"foo", "bar", "baz"},
			expected: path.Root("foo").AtMapKey("bar").AtMapKey("baz"),
		},
		{
			name:     "complex path with mixed types",
			propPath: &[]string{"items", "0", "properties", "name"},
			expected: path.Root("items").AtListIndex(0).AtMapKey("properties").AtMapKey("name"),
		},
		{
			name:     "multiple list indexes",
			propPath: &[]string{"matrix", "2", "3"},
			expected: path.Root("matrix").AtListIndex(2).AtListIndex(3),
		},
		{
			name:     "root with zero index",
			propPath: &[]string{"array", "0"},
			expected: path.Root("array").AtListIndex(0),
		},
		{
			name:     "deep nested structure",
			propPath: &[]string{"config", "servers", "0", "endpoints", "1", "url"},
			expected: path.Root("config").AtMapKey("servers").AtListIndex(0).AtMapKey("endpoints").AtListIndex(1).AtMapKey("url"),
		},
		{
			name:     "numeric string that's not a list index (large number)",
			propPath: &[]string{"data", "99999"},
			expected: path.Root("data").AtListIndex(99999),
		},
		{
			name:     "key that looks like number but isn't",
			propPath: &[]string{"obj", "1abc"},
			expected: path.Root("obj").AtMapKey("1abc"),
		},
		{
			name:     "key with special characters",
			propPath: &[]string{"root", "my-key"},
			expected: path.Root("root").AtMapKey("my-key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PropPathToPath(tt.propPath)

			// For empty path cases
			if tt.expectNil {
				if result.String() != path.Empty().String() {
					t.Errorf("Expected empty path, got %v", result)
				}
				return
			}

			// Compare string representations
			if result.String() != tt.expected.String() {
				t.Errorf("PropPathToPath() = %v, want %v", result.String(), tt.expected.String())
			}
		})
	}
}

// TestPropPathToPath_RealWorldScenarios tests scenarios based on actual Zod validation errors.
func TestPropPathToPath_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		propPath    *[]string
		description string
		expected    path.Path
	}{
		{
			name:        "props validation error",
			propPath:    &[]string{"props", "name"},
			description: "Error on 'name' field in props",
			expected:    path.Root("props").AtMapKey("name"),
		},
		{
			name:        "nested props validation error",
			propPath:    &[]string{"props", "config", "port"},
			description: "Error on nested config.port field",
			expected:    path.Root("props").AtMapKey("config").AtMapKey("port"),
		},
		{
			name:        "array item validation error",
			propPath:    &[]string{"props", "items", "0"},
			description: "Error on first item in array",
			expected:    path.Root("props").AtMapKey("items").AtListIndex(0),
		},
		{
			name:        "nested array item field error",
			propPath:    &[]string{"props", "users", "2", "email"},
			description: "Error on email field of third user",
			expected:    path.Root("props").AtMapKey("users").AtListIndex(2).AtMapKey("email"),
		},
		{
			name:        "state validation error",
			propPath:    &[]string{"state", "status"},
			description: "Error on status field in state",
			expected:    path.Root("state").AtMapKey("status"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PropPathToPath(tt.propPath)

			if result.String() != tt.expected.String() {
				t.Errorf("%s: PropPathToPath() = %v, want %v", tt.description, result.String(), tt.expected.String())
			}
		})
	}
}
