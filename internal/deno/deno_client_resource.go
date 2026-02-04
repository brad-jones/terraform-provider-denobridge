package deno

import (
	"context"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

// DenoClientResource is a client for managing Terraform resources using a Deno runtime.
// It wraps a DenoClient and provides resource-specific functionality for CRUD operations
// (Create, Read, Update, Delete) and plan modification.
type DenoClientResource struct {
	// Client is the underlying Deno client used for JSON-RPC communication
	Client *DenoClient
}

// NewDenoClientResource creates a new DenoClientResource with the specified configuration.
// It initializes a Deno runtime process with the given script and permissions.
//
// Parameters:
//   - denoBinaryPath: The path to the Deno executable
//   - scriptPath: The path to the TypeScript/JavaScript resource script to execute
//   - configPath: The path to the Deno configuration file (deno.json)
//   - permissions: The Deno security permissions to grant the runtime
//
// Returns a configured DenoClientResource ready to manage resources.
func NewDenoClientResource(denoBinaryPath, scriptPath, configPath string, permissions *Permissions) *DenoClientResource {
	return &DenoClientResource{
		NewDenoClient(
			denoBinaryPath,
			scriptPath,
			configPath,
			permissions,
			nil,
		),
	}
}

// CreateRequest represents the request payload for creating a Terraform resource.
// It contains the configuration properties from the Terraform configuration.
type CreateRequest struct {
	// Props contains the resource configuration properties as defined in the Terraform schema
	Props any `json:"props"`
}

// CreateResponse represents the response from creating a Terraform resource.
// It contains the resource's unique identifier and state data.
type CreateResponse struct {
	// ID is the unique identifier for the created resource
	ID string `json:"id"`
	// State contains the resource's state data to be stored in Terraform state
	State any `json:"state"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// Create executes the resource creation operation by calling the "create" method via JSON-RPC.
// It sends the configuration properties to the Deno runtime and retrieves the resource ID and state.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The create request containing the resource configuration properties
//
// Returns the create response containing the resource ID and state, or an error if the JSON-RPC call fails.
func (c *DenoClientResource) Create(ctx context.Context, params *CreateRequest) (*CreateResponse, error) {
	var response *CreateResponse
	if err := c.Client.Socket.Call(ctx, "create", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call create method over JSON-RPC: %v", err)
	}
	return response, nil
}

// CreateReadRequest represents the request payload for reading a Terraform resource.
// It contains the resource ID and configuration properties.
type CreateReadRequest struct {
	// ID is the unique identifier of the resource to read
	ID string `json:"id"`
	// Props contains the resource configuration properties
	Props any `json:"props"`
}

// CreateReadResponse represents the response from reading a Terraform resource.
// It contains the updated properties, state, and existence status of the resource.
type CreateReadResponse struct {
	// Props contains the updated resource properties after reading from the external system
	Props *any `json:"props"`
	// State contains the updated resource state data
	State *any `json:"state"`
	// Exists indicates whether the resource still exists in the external system
	Exists *bool `json:"exists"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// Read executes the resource read operation by calling the "read" method via JSON-RPC.
// It retrieves the current state of the resource from the external system.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The read request containing the resource ID and configuration properties
//
// Returns the read response with updated properties and state, or an error if the JSON-RPC call fails.
func (c *DenoClientResource) Read(ctx context.Context, params *CreateReadRequest) (*CreateReadResponse, error) {
	var response *CreateReadResponse
	if err := c.Client.Socket.Call(ctx, "read", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call read method over JSON-RPC: %v", err)
	}
	return response, nil
}

// UpdateRequest represents the request payload for updating a Terraform resource.
// It contains the resource ID, next configuration, and current configuration and state.
type UpdateRequest struct {
	// ID is the unique identifier of the resource to update
	ID string `json:"id"`
	// NextProps contains the desired resource configuration properties from Terraform
	NextProps any `json:"nextProps"`
	// CurrentProps contains the current resource configuration properties
	CurrentProps any `json:"currentProps"`
	// CurrentState contains the current resource state data
	CurrentState any `json:"currentState"`
}

// UpdateResponse represents the response from updating a Terraform resource.
// It contains the updated resource state data.
type UpdateResponse struct {
	// State contains the updated resource state data after the update operation
	State *any `json:"state"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// Update executes the resource update operation by calling the "update" method via JSON-RPC.
// It sends the desired configuration to the Deno runtime to modify the external resource.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The update request containing the resource ID, next properties, and current state
//
// Returns the update response with the new resource state, or an error if the JSON-RPC call fails.
func (c *DenoClientResource) Update(ctx context.Context, params *UpdateRequest) (*UpdateResponse, error) {
	var response *UpdateResponse
	if err := c.Client.Socket.Call(ctx, "update", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call update method over JSON-RPC: %v", err)
	}
	return response, nil
}

// DeleteRequest represents the request payload for deleting a Terraform resource.
// It contains the resource ID, configuration properties, and state data.
type DeleteRequest struct {
	// ID is the unique identifier of the resource to delete
	ID string `json:"id"`
	// Props contains the resource configuration properties
	Props any `json:"props"`
	// State contains the resource state data
	State any `json:"state"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// DeleteResponse represents the response from deleting a Terraform resource.
// It indicates whether the delete operation completed successfully.
type DeleteResponse struct {
	// Done indicates whether the delete operation completed successfully
	Done bool `json:"done"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// Delete executes the resource deletion operation by calling the "delete" method via JSON-RPC.
// It sends the resource information to the Deno runtime to remove the external resource.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The delete request containing the resource ID, properties, and state
//
// Returns an error if the JSON-RPC call fails or the delete operation is not complete.
func (c *DenoClientResource) Delete(ctx context.Context, params *DeleteRequest) (*DeleteResponse, error) {
	var response *DeleteResponse
	if err := c.Client.Socket.Call(ctx, "delete", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call delete method over JSON-RPC: %v", err)
	}
	return response, nil
}

// ModifyPlanRequest represents the request payload for modifying a Terraform plan.
// It contains the plan type and configuration information for plan customization.
type ModifyPlanRequest struct {
	// ID is the unique identifier of the resource (optional, not present during create operations)
	ID *string `json:"id,omitempty"`
	// PlanType indicates the type of operation being planned ("create", "update", or "delete")
	PlanType string `json:"planType"`
	// NextProps contains the desired resource configuration properties
	NextProps any `json:"nextProps"`
	// CurrentProps contains the current resource configuration properties (not present during create)
	CurrentProps any `json:"currentProps,omitempty"`
	// CurrentState contains the current resource state data (not present during create)
	CurrentState any `json:"currentState,omitempty"`
}

// ModifyPlanResponse represents the response from modifying a Terraform plan.
// It allows the resource to customize the plan, modify properties, or add diagnostics.
type ModifyPlanResponse struct {
	// NoChanges indicates that no changes are required, suppressing the plan
	NoChanges *bool `json:"noChanges,omitempty"`
	// ModifiedProps contains modified property values to be used in the plan
	ModifiedProps *any `json:"modifiedProps,omitempty"`
	// RequiresReplacement indicates that the resource must be replaced (destroy and recreate)
	RequiresReplacement *bool `json:"requiresReplacement,omitempty"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// ModifyPlan executes the plan modification operation by calling the "modifyPlan" method via JSON-RPC.
// It allows the resource to customize the Terraform plan before execution.
// Note: The modifyPlan method is optional; if not implemented in the script, this method returns nil.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The modify plan request containing the plan type and configuration
//
// Returns the modify plan response with plan customizations, or nil if the method is not implemented.
// Returns an error if the JSON-RPC call fails.
func (c *DenoClientResource) ModifyPlan(ctx context.Context, params *ModifyPlanRequest) (*ModifyPlanResponse, error) {
	var response *ModifyPlanResponse
	if err := c.Client.Socket.Call(ctx, "modifyPlan", params, &response); err != nil {

		// ModifyPlan method is optional - return nil if not implemented
		var rpcErr *jsonrpc2.Error
		if errors.As(err, &rpcErr) && rpcErr.Code == jsonrpc2.CodeMethodNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to call modifyPlan method over JSON-RPC: %v", err)
	}

	return response, nil
}
