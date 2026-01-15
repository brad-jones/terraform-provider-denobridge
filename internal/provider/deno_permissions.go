package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type denoPermissions struct {
	All   bool
	Allow []string
	Deny  []string
}

func (permissions *denoPermissions) mapToDenoPermissionsTF() *denoPermissionsTF {
	if permissions == nil {
		return &denoPermissionsTF{
			All:   types.BoolValue(false),
			Allow: types.ListNull(types.StringType),
			Deny:  types.ListNull(types.StringType),
		}
	}

	output := &denoPermissionsTF{
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

type denoPermissionsTF struct {
	All   types.Bool `tfsdk:"all"`
	Allow types.List `tfsdk:"allow"`
	Deny  types.List `tfsdk:"deny"`
}

func (permissions *denoPermissionsTF) mapToDenoPermissions() *denoPermissions {
	if permissions == nil {
		// Default permissions, means deno can not perform any IO of any kind.
		return &denoPermissions{
			All:   false,
			Allow: []string{},
			Deny:  []string{},
		}
	}

	output := &denoPermissions{
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
