package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ action.Action              = &denoBridgeAction{}
	_ action.ActionWithConfigure = &denoBridgeAction{}
)

// NewDenoBridgeAction is a helper function to simplify the provider implementation.
func NewDenoBridgeAction() action.Action {
	return &denoBridgeAction{}
}

// denoBridgeAction defines the action implementation.
type denoBridgeAction struct {
	providerConfig *ProviderConfig
}

// denoBridgeActionModel maps the action schema data.
type denoBridgeActionModel struct {
	Path        types.String       `tfsdk:"path"`
	Props       types.Dynamic      `tfsdk:"props"`
	ConfigFile  types.String       `tfsdk:"config_file"`
	Permissions *denoPermissionsTF `tfsdk:"permissions"`
}

func (a *denoBridgeAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

func (a *denoBridgeAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bridges the terraform-plugin-framework Action to a Deno HTTP Server.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
				Optional:    true,
			},
			"config_file": schema.StringAttribute{
				Description: "File path to a deno config file to use with the deno script. Useful for import maps, etc...",
				Optional:    true,
			},
			"permissions": schema.SingleNestedAttribute{
				Description: "Deno runtime permissions for the script.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"all": schema.BoolAttribute{
						Description: "Grant all permissions.",
						Optional:    true,
					},
					"allow": schema.ListAttribute{
						Description: "List of permissions to allow (e.g., 'read', 'write', 'net').",
						ElementType: types.StringType,
						Optional:    true,
					},
					"deny": schema.ListAttribute{
						Description: "List of permissions to deny.",
						ElementType: types.StringType,
						Optional:    true,
					},
				},
			},
		},
	}
}

func (a *denoBridgeAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.providerConfig = providerConfig
}

func (a *denoBridgeAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	// Read Terraform configuration data into the model
	var data denoBridgeActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno process
	client := NewDenoClient(
		a.providerConfig.DenoBinaryPath,
		data.Path.ValueString(),
		data.ConfigFile.ValueString(),
		data.Permissions.mapToDenoPermissions(),
		"action",
	)
	if err := client.Start(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Failed to start Deno process",
			fmt.Sprintf("Could not start Deno process: %s", err.Error()),
		)
		return
	}
	defer func() {
		if err := client.Stop(); err != nil {
			resp.Diagnostics.AddWarning(
				"Failed to stop Deno process",
				fmt.Sprintf("Could not stop Deno process: %s", err.Error()),
			)
		}
	}()

	// Create JSON-RPC socket
	socket := a.providerConfig.jsocketPackage.NewActionSocket(
		ctx,
		client.GetStdout(),
		client.GetStdin(),
	)
	defer socket.Close()

	// Set up progress handler to send progress events
	socket.SetProgressHandler(func(message string) {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: message,
		})
	})

	// Call the invoke method with the props
	props := fromDynamic(data.Props)
	propsMap, ok := props.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid props type", "Props must be a map")
		return
	}
	result, err := socket.Invoke(ctx, propsMap)
	if err != nil {
		resp.Diagnostics.AddError("invoke failed", err.Error())
		return
	}

	// Store the result (optional)
	_ = result
}
