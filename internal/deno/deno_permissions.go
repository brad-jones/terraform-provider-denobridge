package deno

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Permissions represents Deno runtime security permissions in Go-native types.
// It controls what system resources the Deno runtime can access during execution.
type Permissions struct {
	// All grants all permissions when true, effectively disabling security restrictions
	All bool
	// Allow is a list of specific permissions to grant (e.g., "read", "write", "net", "env")
	Allow []string
	// Deny is a list of specific permissions to explicitly deny
	Deny []string
}

// MapToDenoPermissionsTF converts Go-native Permissions to Terraform Framework types.
// This is used when returning permission data to Terraform state or configuration.
//
// If permissions is nil, returns a PermissionsTF with safe defaults (All=false, empty lists).
//
// Returns a PermissionsTF struct with types.Bool and types.List fields suitable for Terraform.
func (permissions *Permissions) MapToDenoPermissionsTF() *PermissionsTF {
	if permissions == nil {
		return &PermissionsTF{
			All:   types.BoolValue(false),
			Allow: types.ListNull(types.StringType),
			Deny:  types.ListNull(types.StringType),
		}
	}

	output := &PermissionsTF{
		All: types.BoolValue(permissions.All),
	}

	// Convert Allow []string to types.List
	if len(permissions.Allow) == 0 {
		output.Allow = types.ListValueMust(types.StringType, []attr.Value{})
	} else {
		allowElements := make([]attr.Value, 0, len(permissions.Allow))
		for _, allow := range permissions.Allow {
			allowElements = append(allowElements, types.StringValue(allow))
		}
		output.Allow = types.ListValueMust(types.StringType, allowElements)
	}

	// Convert Deny []string to types.List
	if len(permissions.Deny) == 0 {
		output.Deny = types.ListValueMust(types.StringType, []attr.Value{})
	} else {
		denyElements := make([]attr.Value, 0, len(permissions.Deny))
		for _, deny := range permissions.Deny {
			denyElements = append(denyElements, types.StringValue(deny))
		}
		output.Deny = types.ListValueMust(types.StringType, denyElements)
	}

	return output
}

// PermissionsTF represents Deno runtime security permissions using Terraform Framework types.
// This struct is used for schema definitions and state management in Terraform.
type PermissionsTF struct {
	// All grants all permissions when true, effectively disabling security restrictions
	All types.Bool `tfsdk:"all"`
	// Allow is a list of specific permissions to grant (e.g., "read", "write", "net", "env")
	Allow types.List `tfsdk:"allow"`
	// Deny is a list of specific permissions to explicitly deny
	Deny types.List `tfsdk:"deny"`
}

// MapToDenoPermissions converts Terraform Framework types to Go-native Permissions.
// This is used when reading permission configuration from Terraform into Go code.
//
// If permissions is nil, returns safe default permissions (All=false, empty slices),
// which means the Deno runtime cannot perform any I/O operations.
//
// Returns a Permissions struct with native Go types (bool and []string).
func (permissions *PermissionsTF) MapToDenoPermissions() *Permissions {
	if permissions == nil {
		// Default permissions, means deno can not perform any IO of any kind.
		return &Permissions{
			All:   false,
			Allow: []string{},
			Deny:  []string{},
		}
	}

	output := &Permissions{
		All: permissions.All.ValueBool(),
	}

	if !permissions.Allow.IsNull() {
		allowElements := permissions.Allow.Elements()
		output.Allow = make([]string, 0, len(allowElements))
		for _, elem := range allowElements {
			if strVal, ok := elem.(types.String); ok {
				output.Allow = append(output.Allow, strVal.ValueString())
			}
		}
	}

	if !permissions.Deny.IsNull() {
		denyElements := permissions.Deny.Elements()
		output.Deny = make([]string, 0, len(denyElements))
		for _, elem := range denyElements {
			if strVal, ok := elem.(types.String); ok {
				output.Deny = append(output.Deny, strVal.ValueString())
			}
		}
	}

	return output
}
