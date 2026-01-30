package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &denoBridgeResource{}
	_ resource.ResourceWithConfigure   = &denoBridgeResource{}
	_ resource.ResourceWithModifyPlan  = &denoBridgeResource{}
	_ resource.ResourceWithImportState = &denoBridgeResource{}
)

// NewDenoBridgeResource is a helper function to simplify the provider implementation.
func NewDenoBridgeResource() resource.Resource {
	return &denoBridgeResource{}
}

// denoBridgeResource is the resource implementation.
type denoBridgeResource struct {
	providerConfig *ProviderConfig
}

// denoBridgeResourceModel maps the resource schema data.
type denoBridgeResourceModel struct {
	ID          types.String       `tfsdk:"id"`
	Path        types.String       `tfsdk:"path"`
	Props       types.Dynamic      `tfsdk:"props"`
	State       types.Dynamic      `tfsdk:"state"`
	ConfigFile  types.String       `tfsdk:"config_file"`
	Permissions *denoPermissionsTF `tfsdk:"permissions"`
}

// Metadata returns the resource type name.
func (r *denoBridgeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

// Schema defines the schema for the resource.
func (r *denoBridgeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bridges the terraform-plugin-framework Resource to a Deno HTTP Server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
				Optional:    true,
			},
			"state": schema.DynamicAttribute{
				Description: "Additional computed state of the resource as returned by the Deno script.",
				Computed:    true,
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

// Configure adds the provider configured client to the resource.
func (r *denoBridgeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.providerConfig = providerConfig
}

// Create creates the resource and sets the initial Terraform state.
func (r *denoBridgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan denoBridgeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno process
	client := NewDenoClient(
		r.providerConfig.DenoBinaryPath,
		plan.Path.ValueString(),
		plan.ConfigFile.ValueString(),
		plan.Permissions.mapToDenoPermissions(),
		"resource",
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
	socket := r.providerConfig.jsocketPackage.NewResourceSocket(
		ctx,
		client.GetStdout(),
		client.GetStdin(),
	)
	defer socket.Close()

	// Call the create method
	props := fromDynamic(plan.Props)
	propsMap, ok := props.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid props type", "Props must be a map")
		return
	}
	id, state, err := socket.Create(ctx, propsMap)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create resource",
			fmt.Sprintf("Could not create resource via Deno script: %s", err.Error()),
		)
		return
	}

	// Set state - convert id to string
	idStr, ok := id.(string)
	if !ok {
		// Try to convert other types to string
		idStr = fmt.Sprintf("%v", id)
	}
	plan.ID = types.StringValue(idStr)
	plan.State = toDynamic(state)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *denoBridgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state denoBridgeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno process
	client := NewDenoClient(
		r.providerConfig.DenoBinaryPath,
		state.Path.ValueString(),
		state.ConfigFile.ValueString(),
		state.Permissions.mapToDenoPermissions(),
		"resource",
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
	socket := r.providerConfig.jsocketPackage.NewResourceSocket(
		ctx,
		client.GetStdout(),
		client.GetStdin(),
	)
	defer socket.Close()

	// Call the read method
	props := fromDynamic(state.Props)
	propsMap, ok := props.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid props type", "Props must be a map")
		return
	}
	response, err := socket.Read(ctx, state.ID.ValueString(), propsMap)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read resource",
			fmt.Sprintf("Could not read resource via Deno script: %s", err.Error()),
		)
		return
	}

	if response.Exists != nil && !*response.Exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Set refreshed state
	state.Props = toDynamic(response.Props)
	state.State = toDynamic(response.State)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *denoBridgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan denoBridgeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state for ID
	var state denoBridgeResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno process
	client := NewDenoClient(
		r.providerConfig.DenoBinaryPath,
		plan.Path.ValueString(),
		plan.ConfigFile.ValueString(),
		plan.Permissions.mapToDenoPermissions(),
		"resource",
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
	socket := r.providerConfig.jsocketPackage.NewResourceSocket(
		ctx,
		client.GetStdout(),
		client.GetStdin(),
	)
	defer socket.Close()

	// Call the update method
	nextProps := fromDynamic(plan.Props)
	nextPropsMap, ok := nextProps.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid props type", "Props must be a map")
		return
	}
	currentProps := fromDynamic(state.Props)
	currentPropsMap, ok := currentProps.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid props type", "Props must be a map")
		return
	}
	currentState := fromDynamic(state.State)
	currentStateMap, ok := currentState.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid state type", "State must be a map")
		return
	}
	updatedState, err := socket.Update(
		ctx,
		state.ID.ValueString(),
		nextPropsMap,
		currentPropsMap,
		currentStateMap,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update resource",
			fmt.Sprintf("Could not update resource via Deno script: %s", err.Error()),
		)
		return
	}

	// Keep the same ID
	plan.ID = state.ID

	// Set updated state
	plan.State = toDynamic(updatedState)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *denoBridgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state denoBridgeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno process
	client := NewDenoClient(
		r.providerConfig.DenoBinaryPath,
		state.Path.ValueString(),
		state.ConfigFile.ValueString(),
		state.Permissions.mapToDenoPermissions(),
		"resource",
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
	socket := r.providerConfig.jsocketPackage.NewResourceSocket(
		ctx,
		client.GetStdout(),
		client.GetStdin(),
	)
	defer socket.Close()

	// Call the delete method
	props := fromDynamic(state.Props)
	propsMap, ok := props.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid props type", "Props must be a map")
		return
	}
	stateData := fromDynamic(state.State)
	stateMap, ok := stateData.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Invalid state type", "State must be a map")
		return
	}
	if err := socket.Delete(ctx, state.ID.ValueString(), propsMap, stateMap); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete resource",
			fmt.Sprintf("Could not delete resource via Deno script: %s", err.Error()),
		)
		return
	}
}

// ModifyPlan calls the Deno script's optional /modify-plan endpoint to allow custom plan modification.
// The script can return modified props, specify attributes requiring replacement, and add diagnostics.
func (r *denoBridgeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get the plan data, if it exists
	var plan *denoBridgeResourceModel
	if !req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Get the state data, if it exists
	var state *denoBridgeResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Bail out early if nothing is actually changing for updates
	if plan != nil && state != nil {
		if plan.Props.Equal(state.Props) {
			return
		}
	}

	// Get the deno script from the plan for create & update operations.
	// Otherwise for delete we get the details from the existing state.
	var denoScriptPath string
	var denoConfigPath string
	var denoPermissions *denoPermissionsTF
	if plan != nil {
		denoScriptPath = plan.Path.ValueString()
		denoConfigPath = plan.ConfigFile.ValueString()
		denoPermissions = plan.Permissions
	} else {
		if state != nil {
			denoScriptPath = state.Path.ValueString()
			denoConfigPath = state.ConfigFile.ValueString()
			denoPermissions = state.Permissions
		}
	}

	// Bail out if we can't call deno
	if denoScriptPath == "" || denoPermissions == nil {
		resp.Diagnostics.AddWarning("ModifyPlan SKIPPED", "missing denoScriptPath or denoPermissions")
		return
	}

	// Start the Deno process
	client := NewDenoClient(
		r.providerConfig.DenoBinaryPath,
		denoScriptPath,
		denoConfigPath,
		denoPermissions.mapToDenoPermissions(),
		"resource",
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
	socket := r.providerConfig.jsocketPackage.NewResourceSocket(
		ctx,
		client.GetStdout(),
		client.GetStdin(),
	)
	defer socket.Close()

	// Build the request parameters
	var id any
	if state != nil {
		id = state.ID.ValueString()
	}
	var nextPropsMap map[string]any
	var currentPropsMap map[string]any
	var currentStateMap map[string]any
	if plan != nil && state == nil {
		// Create plan
		nextProps := fromDynamic(plan.Props)
		if m, ok := nextProps.(map[string]any); ok {
			nextPropsMap = m
		}
	}
	if plan != nil && state != nil {
		// Update plan
		nextProps := fromDynamic(plan.Props)
		if m, ok := nextProps.(map[string]any); ok {
			nextPropsMap = m
		}
		currentProps := fromDynamic(state.Props)
		if m, ok := currentProps.(map[string]any); ok {
			currentPropsMap = m
		}
		currentState := fromDynamic(state.State)
		if m, ok := currentState.(map[string]any); ok {
			currentStateMap = m
		}
	}
	if plan == nil && state != nil {
		// Delete plan
		currentProps := fromDynamic(state.Props)
		if m, ok := currentProps.(map[string]any); ok {
			currentPropsMap = m
		}
		currentState := fromDynamic(state.State)
		if m, ok := currentState.(map[string]any); ok {
			currentStateMap = m
		}
	}

	// Call the modifyPlan method (optional)
	responsePayload, err := socket.ModifyPlan(ctx, id, nextPropsMap, currentPropsMap, currentStateMap)
	if err != nil {
		// Check if method not found (optional method)
		if err.Error() == "Method not found" {
			return
		}
		resp.Diagnostics.AddError("modifyPlan failed", err.Error())
		return
	}

	// Handle modified props - allows the script to modify the planned properties
	if len(responsePayload.ModifiedProps) > 0 {
		plan.Props = toDynamic(&responsePayload.ModifiedProps)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}

	// Handle requiresReplacement - instructing tf to do a create then delete instead of an update
	if responsePayload.RequiresReplacement != nil && *responsePayload.RequiresReplacement {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("props"))
	}

	// Handle diagnostics - allows the script to add warnings or errors
	// Mainly for use with the Resource Destroy Plan Diagnostics workflow.
	// see: https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#resource-destroy-plan-diagnostics
	if len(responsePayload.Diagnostics) > 0 {
		for _, diag := range responsePayload.Diagnostics {
			switch diag.Severity {
			case "error":
				resp.Diagnostics.AddError(diag.Summary, diag.Detail)
			case "warning":
				resp.Diagnostics.AddWarning(diag.Summary, diag.Detail)
			}
		}
	}
}

// ImportState imports an existing resource into Terraform state.
// The import ID must be a JSON string containing the resource ID, Deno script path,
// and any required permissions. Props are optional and should only include properties
// needed to uniquely identify the resource (resource-dependent).
func (r *denoBridgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var importConfig struct {
		ID          string           `json:"id"`
		Path        string           `json:"path"`
		Props       *map[string]any  `json:"props,omitempty"`
		ConfigFile  *string          `json:"config_file,omitempty"`
		Permissions *denoPermissions `json:"permissions,omitempty"`
	}
	err := json.Unmarshal([]byte(req.ID), &importConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Import ID must be valid JSON containing id, path, and optional props/permissions. Error: %s", err.Error()),
		)
		return
	}

	var props types.Dynamic
	if importConfig.Props != nil {
		props = toDynamic(importConfig.Props)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, denoBridgeResourceModel{
		ID:          types.StringValue(importConfig.ID),
		Path:        types.StringValue(importConfig.Path),
		Props:       props,
		ConfigFile:  types.StringPointerValue(importConfig.ConfigFile),
		Permissions: importConfig.Permissions.mapToDenoPermissionsTF(),
	})...)
}
