package jsocket

import (
	"context"
	"fmt"
	"io"

	"github.com/sourcegraph/jsonrpc2"
)

// Common types used across all sockets

// Diagnostic represents a plan-time diagnostic message
type Diagnostic struct {
	Severity string `json:"severity"` // "error" or "warning"
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
}

// DatasourceSocket provides strongly-typed JSON-RPC methods for data source operations
type DatasourceSocket struct {
	socket *JSocket
}

// NewDatasourceSocket creates a new datasource socket with no incoming RPC handlers
func NewDatasourceSocket(ctx context.Context, reader io.ReadCloser, writer io.Writer, opts ...jsonrpc2.ConnOpt) *DatasourceSocket {
	return &DatasourceSocket{
		socket: New(ctx, reader, writer, func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
			// Data sources don't receive any RPCs from Deno, only send them
			return map[string]any{}
		}, opts...),
	}
}

// ReadRequest represents the parameters for a data source read operation
type ReadRequest struct {
	Props map[string]any `json:"props"`
}

// ReadResponse represents the result of a data source read operation
type ReadResponse struct {
	Result map[string]any `json:"result"`
}

// Read calls the data source's read method
func (s *DatasourceSocket) Read(ctx context.Context, props map[string]any) (map[string]any, error) {
	var response ReadResponse
	if err := s.socket.Call(ctx, "read", ReadRequest{Props: props}, &response); err != nil {
		return nil, fmt.Errorf("failed to call read: %w", err)
	}
	return response.Result, nil
}

// Close closes the socket connection
func (s *DatasourceSocket) Close() error {
	return s.socket.Close()
}

// ResourceSocket provides strongly-typed JSON-RPC methods for resource operations
type ResourceSocket struct {
	socket *JSocket
}

// NewResourceSocket creates a new resource socket with no incoming RPC handlers
func NewResourceSocket(ctx context.Context, reader io.ReadCloser, writer io.Writer, opts ...jsonrpc2.ConnOpt) *ResourceSocket {
	return &ResourceSocket{
		socket: New(ctx, reader, writer, func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
			// Resources don't receive any RPCs from Deno, only send them
			return map[string]any{}
		}, opts...),
	}
}

// CreateRequest represents the parameters for a resource create operation
type CreateRequest struct {
	Props map[string]any `json:"props"`
}

// CreateResponse represents the result of a resource create operation
type CreateResponse struct {
	ID    any            `json:"id"`
	State map[string]any `json:"state"`
}

// Create calls the resource's create method
func (s *ResourceSocket) Create(ctx context.Context, props map[string]any) (any, map[string]any, error) {
	var response CreateResponse
	if err := s.socket.Call(ctx, "create", CreateRequest{Props: props}, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to call create: %w", err)
	}
	return response.ID, response.State, nil
}

// ResourceReadRequest represents the parameters for a resource read operation
type ResourceReadRequest struct {
	ID    any            `json:"id"`
	Props map[string]any `json:"props"`
}

// ResourceReadResponse represents the result of a resource read operation
type ResourceReadResponse struct {
	Props  map[string]any `json:"props,omitempty"`
	State  map[string]any `json:"state,omitempty"`
	Exists *bool          `json:"exists,omitempty"`
}

// Read calls the resource's read method
func (s *ResourceSocket) Read(ctx context.Context, id any, props map[string]any) (*ResourceReadResponse, error) {
	var response ResourceReadResponse
	if err := s.socket.Call(ctx, "read", ResourceReadRequest{ID: id, Props: props}, &response); err != nil {
		return nil, fmt.Errorf("failed to call read: %w", err)
	}
	return &response, nil
}

// UpdateRequest represents the parameters for a resource update operation
type UpdateRequest struct {
	ID           any            `json:"id"`
	NextProps    map[string]any `json:"nextProps"`
	CurrentProps map[string]any `json:"currentProps"`
	CurrentState map[string]any `json:"currentState"`
}

// UpdateResponse represents the result of a resource update operation
type UpdateResponse struct {
	State map[string]any `json:"state"`
}

// Update calls the resource's update method
func (s *ResourceSocket) Update(ctx context.Context, id any, nextProps, currentProps, currentState map[string]any) (map[string]any, error) {
	var response UpdateResponse
	if err := s.socket.Call(ctx, "update", UpdateRequest{
		ID:           id,
		NextProps:    nextProps,
		CurrentProps: currentProps,
		CurrentState: currentState,
	}, &response); err != nil {
		return nil, fmt.Errorf("failed to call update: %w", err)
	}
	return response.State, nil
}

// DeleteRequest represents the parameters for a resource delete operation
type DeleteRequest struct {
	ID    any            `json:"id"`
	Props map[string]any `json:"props"`
	State map[string]any `json:"state"`
}

// Delete calls the resource's delete method
func (s *ResourceSocket) Delete(ctx context.Context, id any, props, state map[string]any) error {
	// Delete returns no response
	var response any
	if err := s.socket.Call(ctx, "delete", DeleteRequest{ID: id, Props: props, State: state}, &response); err != nil {
		return fmt.Errorf("failed to call delete: %w", err)
	}
	return nil
}

// ModifyPlanRequest represents the parameters for a resource plan modification
type ModifyPlanRequest struct {
	ID           any            `json:"id"` // null for create
	NextProps    map[string]any `json:"nextProps"`
	CurrentProps map[string]any `json:"currentProps,omitempty"`
	CurrentState map[string]any `json:"currentState,omitempty"`
}

// ModifyPlanResponse represents the result of a plan modification
type ModifyPlanResponse struct {
	ModifiedProps       map[string]any `json:"modifiedProps,omitempty"`
	RequiresReplacement *bool          `json:"requiresReplacement,omitempty"`
	Diagnostics         []Diagnostic   `json:"diagnostics,omitempty"`
}

// ModifyPlan calls the resource's optional modifyPlan method
// Returns jsonrpc2.CodeMethodNotFound error if not implemented
func (s *ResourceSocket) ModifyPlan(ctx context.Context, id any, nextProps, currentProps, currentState map[string]any) (*ModifyPlanResponse, error) {
	var response ModifyPlanResponse
	if err := s.socket.Call(ctx, "modifyPlan", ModifyPlanRequest{
		ID:           id,
		NextProps:    nextProps,
		CurrentProps: currentProps,
		CurrentState: currentState,
	}, &response); err != nil {
		return nil, err // Return error as-is so caller can check for CodeMethodNotFound
	}
	return &response, nil
}

// Close closes the socket connection
func (s *ResourceSocket) Close() error {
	return s.socket.Close()
}

// ActionSocket provides strongly-typed JSON-RPC methods for action operations
type ActionSocket struct {
	socket             *JSocket
	progressHandler    func(message string)
	progressHandlerSet bool
}

// NewActionSocket creates a new action socket with incoming RPC handler for progress
func NewActionSocket(ctx context.Context, reader io.ReadCloser, writer io.Writer, opts ...jsonrpc2.ConnOpt) *ActionSocket {
	actionSocket := &ActionSocket{}

	actionSocket.socket = New(ctx, reader, writer, func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
		return map[string]any{
			"invokeProgress": func(params struct{ Message string }) {
				if actionSocket.progressHandlerSet && actionSocket.progressHandler != nil {
					actionSocket.progressHandler(params.Message)
				}
			},
		}
	}, opts...)

	return actionSocket
}

// SetProgressHandler sets the handler for progress notifications from the action
func (s *ActionSocket) SetProgressHandler(handler func(message string)) {
	s.progressHandler = handler
	s.progressHandlerSet = true
}

// InvokeRequest represents the parameters for an action invocation
type InvokeRequest struct {
	Props map[string]any `json:"props"`
}

// InvokeResponse represents the result of an action invocation
type InvokeResponse struct {
	Result map[string]any `json:"result"`
}

// Invoke calls the action's invoke method
func (s *ActionSocket) Invoke(ctx context.Context, props map[string]any) (map[string]any, error) {
	var response InvokeResponse
	if err := s.socket.Call(ctx, "invoke", InvokeRequest{Props: props}, &response); err != nil {
		return nil, fmt.Errorf("failed to call invoke: %w", err)
	}
	return response.Result, nil
}

// Close closes the socket connection
func (s *ActionSocket) Close() error {
	return s.socket.Close()
}

// EphemeralResourceSocket provides strongly-typed JSON-RPC methods for ephemeral resource operations
type EphemeralResourceSocket struct {
	socket *JSocket
}

// NewEphemeralResourceSocket creates a new ephemeral resource socket with no incoming RPC handlers
func NewEphemeralResourceSocket(ctx context.Context, reader io.ReadCloser, writer io.Writer, opts ...jsonrpc2.ConnOpt) *EphemeralResourceSocket {
	return &EphemeralResourceSocket{
		socket: New(ctx, reader, writer, func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
			// Ephemeral resources don't receive any RPCs from Deno, only send them
			return map[string]any{}
		}, opts...),
	}
}

// OpenRequest represents the parameters for opening an ephemeral resource
type OpenRequest struct {
	Props map[string]any `json:"props"`
}

// OpenResponse represents the result of opening an ephemeral resource
type OpenResponse struct {
	Result  map[string]any `json:"result"`
	RenewAt *int64         `json:"renewAt,omitempty"`
	Private map[string]any `json:"private,omitempty"`
}

// Open calls the ephemeral resource's open method
func (s *EphemeralResourceSocket) Open(ctx context.Context, props map[string]any) (*OpenResponse, error) {
	var response OpenResponse
	if err := s.socket.Call(ctx, "open", OpenRequest{Props: props}, &response); err != nil {
		return nil, fmt.Errorf("failed to call open: %w", err)
	}
	return &response, nil
}

// RenewRequest represents the parameters for renewing an ephemeral resource
type RenewRequest struct {
	Private map[string]any `json:"private"`
}

// RenewResponse represents the result of renewing an ephemeral resource
type RenewResponse struct {
	RenewAt *int64         `json:"renewAt,omitempty"`
	Private map[string]any `json:"private,omitempty"`
}

// Renew calls the ephemeral resource's optional renew method
// Returns jsonrpc2.CodeMethodNotFound error if not implemented
func (s *EphemeralResourceSocket) Renew(ctx context.Context, private map[string]any) (*RenewResponse, error) {
	var response RenewResponse
	if err := s.socket.Call(ctx, "renew", RenewRequest{Private: private}, &response); err != nil {
		return nil, err // Return error as-is so caller can check for CodeMethodNotFound
	}
	return &response, nil
}

// CloseRequest represents the parameters for closing an ephemeral resource
type CloseRequest struct {
	Private map[string]any `json:"private"`
}

// CloseResource calls the ephemeral resource's optional close method
// Returns jsonrpc2.CodeMethodNotFound error if not implemented
func (s *EphemeralResourceSocket) CloseResource(ctx context.Context, private map[string]any) error {
	var response any
	if err := s.socket.Call(ctx, "close", CloseRequest{Private: private}, &response); err != nil {
		return err // Return error as-is so caller can check for CodeMethodNotFound
	}
	return nil
}

// Close closes the socket connection
func (s *EphemeralResourceSocket) Close() error {
	return s.socket.Close()
}
