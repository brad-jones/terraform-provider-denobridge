package deno

import (
	"context"
	"fmt"
)

// DenoClientDatasource is a client for reading Terraform data sources using a Deno runtime.
// It wraps a DenoClient and provides data source-specific functionality for reading
// external data into Terraform configurations.
type DenoClientDatasource struct {
	// Client is the underlying Deno client used for JSON-RPC communication
	Client *DenoClient
}

// NewDenoClientDatasource creates a new DenoClientDatasource with the specified configuration.
// It initializes a Deno runtime process with the given script and permissions.
//
// Parameters:
//   - denoBinaryPath: The path to the Deno executable
//   - scriptPath: The path to the TypeScript/JavaScript data source script to execute
//   - configPath: The path to the Deno configuration file (deno.json)
//   - permissions: The Deno security permissions to grant the runtime
//
// Returns a configured DenoClientDatasource ready to read data.
func NewDenoClientDatasource(denoBinaryPath, scriptPath, configPath string, permissions *Permissions) *DenoClientDatasource {
	return &DenoClientDatasource{
		NewDenoClient(
			denoBinaryPath,
			scriptPath,
			configPath,
			permissions,
			nil,
		),
	}
}

// ReadRequest represents the request payload for reading a Terraform data source.
// It contains the configuration properties passed to the data source from the Terraform configuration.
type ReadRequest struct {
	// Props contains the data source configuration properties as defined in the Terraform schema
	Props any `json:"props"`
}

// ReadResponse represents the response from reading a Terraform data source.
// It contains the data retrieved from the external source.
type ReadResponse struct {
	// Result contains the data returned by the data source, which will be stored in Terraform state
	Result any `json:"result"`
	// SensitiveResult contains the data source sensitive data (marked as sensitive in Terraform)
	SensitiveResult any `json:"sensitiveResult"`
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

// Read executes the data source read operation by calling the "read" method via JSON-RPC.
// It sends the configuration properties to the Deno runtime and retrieves the resulting data.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The read request containing the data source configuration properties
//
// Returns the read response containing the retrieved data, or an error if the JSON-RPC call fails.
func (c *DenoClientDatasource) Read(ctx context.Context, params *ReadRequest) (*ReadResponse, error) {
	var response *ReadResponse
	if err := c.Client.Socket.Call(ctx, "read", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call read method over JSON-RPC: %v", err)
	}
	return response, nil
}
