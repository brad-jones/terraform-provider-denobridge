# JSON-RPC Interface

The terraform-provider-denobridge communicates with your TypeScript/JavaScript code using [JSON-RPC 2.0](https://www.jsonrpc.org/specification) over standard input/output (stdin/stdout). This document describes the low-level protocol interface using the [OpenRPC](https://open-rpc.org/) specification format.

For most use cases, you should use the provided abstract base classes from the `@brad-jones/terraform-provider-denobridge` JSR package rather than implementing the JSON-RPC protocol directly.

## Transport

- **Protocol**: JSON-RPC 2.0
- **Transport**: stdio (standard input/output)
- **Encoding**: Newline-delimited JSON (each message is a single JSON object followed by `\n`)
- **Direction**: Bidirectional - both Go and TypeScript can send requests and notifications

## Debug Logging

Set the `TF_LOG` environment variable to `DEBUG` to enable verbose JSON-RPC logging to stderr:

```bash
TF_LOG=DEBUG terraform apply
```

## Error Codes

The provider uses standard JSON-RPC 2.0 error codes:

- **-32700**: Parse error - Invalid JSON
- **-32600**: Invalid request - Missing required fields
- **-32601**: Method not found - Used for optional methods (modifyPlan, renew, close)
- **-32602**: Invalid params - Parameter validation failed
- **-32603**: Internal error - Unexpected error in handler

## Provider Types

The provider generates different entrypoint scripts based on the provider type:

### Data Source

Implements read-only data fetching operations.

**Methods**: `read`

### Resource

Implements full CRUD lifecycle management.

**Methods**: `create`, `read`, `update`, `delete`, `modifyPlan` (optional)

### Action

Implements one-time operations with progress reporting.

**Methods**: `invoke`
**Notifications**: `invokeProgress` (sent from TypeScript to Go)

### Ephemeral Resource

Implements temporary resources with optional renewal and cleanup.

**Methods**: `open`, `renew` (optional), `close` (optional)

---

## OpenRPC Specification

### Data Source Methods

#### read

Fetches data from an external source based on the provided properties.

**Parameters**:

```json
{
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Input properties from the Terraform configuration"
  }
}
```

**Result**:

```json
{
  "result": {
    "type": "object",
    "additionalProperties": true,
    "description": "Output data to be stored in Terraform state"
  }
}
```

**Example Request**:

```json
{ "jsonrpc": "2.0", "id": 1, "method": "read", "params": { "props": { "domain": "example.com", "record_type": "A" } } }
```

**Example Response**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "result": { "ip": "93.184.216.34", "ttl": 3600 } } }
```

---

### Resource Methods

#### create

Creates a new resource instance.

**Parameters**:

```json
{
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Input properties from the Terraform configuration"
  }
}
```

**Result**:

```json
{
  "id": {
    "type": ["string", "number", "object"],
    "description": "Unique identifier for the resource (can be string, number, or complex object)"
  },
  "state": {
    "type": "object",
    "additionalProperties": true,
    "description": "Computed state data (read-only values, internal tracking data)"
  }
}
```

**Example Request**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "create",
  "params": { "props": { "path": "/tmp/test.txt", "content": "Hello World" } }
}
```

**Example Response**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { "id": "/tmp/test.txt", "state": { "size": 11, "created_at": "2026-01-30T12:00:00Z" } }
}
```

#### read

Reads the current state of an existing resource.

**Parameters**:

```json
{
  "id": {
    "type": ["string", "number", "object"],
    "description": "Resource identifier from create or import"
  },
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current input properties from Terraform state"
  }
}
```

**Result**:

```json
{
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Updated properties (if drift detected)",
    "optional": true
  },
  "state": {
    "type": "object",
    "additionalProperties": true,
    "description": "Updated computed state",
    "optional": true
  },
  "exists": {
    "type": "boolean",
    "description": "Whether the resource still exists (defaults to true if omitted)",
    "optional": true
  }
}
```

**Example Request**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "read",
  "params": { "id": "/tmp/test.txt", "props": { "path": "/tmp/test.txt", "content": "Hello World" } }
}
```

**Example Response (exists)**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "state": { "size": 11, "modified_at": "2026-01-30T12:05:00Z" } } }
```

**Example Response (deleted outside Terraform)**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "exists": false } }
```

#### update

Updates an existing resource with new properties.

**Parameters**:

```json
{
  "id": {
    "type": ["string", "number", "object"],
    "description": "Resource identifier"
  },
  "nextProps": {
    "type": "object",
    "additionalProperties": true,
    "description": "New desired properties from Terraform configuration"
  },
  "currentProps": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current properties from Terraform state"
  },
  "currentState": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current computed state from Terraform state"
  }
}
```

**Result**:

```json
{
  "state": {
    "type": "object",
    "additionalProperties": true,
    "description": "Updated computed state after applying changes"
  }
}
```

**Example Request**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "update",
  "params": {
    "id": "/tmp/test.txt",
    "nextProps": { "path": "/tmp/test.txt", "content": "Updated content" },
    "currentProps": { "path": "/tmp/test.txt", "content": "Hello World" },
    "currentState": { "size": 11 }
  }
}
```

**Example Response**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "state": { "size": 15, "modified_at": "2026-01-30T12:10:00Z" } } }
```

#### delete

Deletes an existing resource.

**Parameters**:

```json
{
  "id": {
    "type": ["string", "number", "object"],
    "description": "Resource identifier"
  },
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current properties from Terraform state"
  },
  "state": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current computed state from Terraform state"
  }
}
```

**Result**: No response body (returns `null`)

**Example Request**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "delete",
  "params": {
    "id": "/tmp/test.txt",
    "props": { "path": "/tmp/test.txt", "content": "Hello World" },
    "state": { "size": 11 }
  }
}
```

**Example Response**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": null }
```

#### modifyPlan (optional)

Modifies the Terraform plan before it's shown to the user. This is called during `terraform plan` and `terraform apply` before actual changes are made.

**Error Handling**: If this method is not implemented, return error code `-32601` (Method not found). The provider will treat this as acceptable and proceed without plan modification.

**Parameters**:

```json
{
  "id": {
    "type": ["string", "number", "object", "null"],
    "description": "Resource identifier (null for create operations)"
  },
  "nextProps": {
    "type": "object",
    "additionalProperties": true,
    "description": "Planned properties (create/update) or null (delete)",
    "optional": true
  },
  "currentProps": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current properties (update/delete) or null (create)",
    "optional": true
  },
  "currentState": {
    "type": "object",
    "additionalProperties": true,
    "description": "Current state (update/delete) or null (create)",
    "optional": true
  }
}
```

**Result**:

```json
{
  "modifiedProps": {
    "type": "object",
    "additionalProperties": true,
    "description": "Modified properties to use in the plan (if you want to change them)",
    "optional": true
  },
  "requiresReplacement": {
    "type": "boolean",
    "description": "If true, forces resource replacement (delete + create) instead of update",
    "optional": true
  },
  "diagnostics": {
    "type": "array",
    "items": {
      "type": "object",
      "properties": {
        "severity": {
          "type": "string",
          "enum": ["error", "warning"],
          "description": "Severity level of the diagnostic"
        },
        "summary": {
          "type": "string",
          "description": "Short summary of the diagnostic"
        },
        "detail": {
          "type": "string",
          "description": "Detailed explanation of the diagnostic"
        }
      }
    },
    "description": "Warnings or errors to display during plan",
    "optional": true
  }
}
```

**Example Request (create plan)**:

```json
{ "jsonrpc": "2.0", "id": 1, "method": "modifyPlan", "params": { "id": null, "nextProps": { "name": "test" } } }
```

**Example Response (add default value)**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "modifiedProps": { "name": "test", "size": 100 } } }
```

**Example Response (require replacement)**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "requiresReplacement": true } }
```

**Example Response (not implemented)**:

```json
{ "jsonrpc": "2.0", "id": 1, "error": { "code": -32601, "message": "Method not found" } }
```

---

### Action Methods

#### invoke

Executes a one-time operation with optional progress reporting.

**Parameters**:

```json
{
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Input properties for the action"
  }
}
```

**Result**:

```json
{
  "result": {
    "type": "object",
    "additionalProperties": true,
    "description": "Output data from the action execution"
  }
}
```

**Example Request**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "invoke",
  "params": { "props": { "command": "deploy", "target": "production" } }
}
```

**Example Response**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "result": { "deployed": true, "version": "v1.2.3" } } }
```

#### invokeProgress (notification)

Sent from TypeScript to Go to report progress during action execution. This is a **notification** (no response expected).

**Direction**: TypeScript → Go

**Parameters**:

```json
{
  "message": {
    "type": "string",
    "description": "Progress message to display to the user"
  }
}
```

**Example Notification**:

```json
{ "jsonrpc": "2.0", "method": "invokeProgress", "params": { "message": "Step 1/3: Validating configuration..." } }
```

---

### Ephemeral Resource Methods

#### open

Opens an ephemeral resource and optionally schedules automatic renewal.

**Parameters**:

```json
{
  "props": {
    "type": "object",
    "additionalProperties": true,
    "description": "Input properties for opening the resource"
  }
}
```

**Result**:

```json
{
  "result": {
    "type": "object",
    "additionalProperties": true,
    "description": "Output data (credentials, tokens, etc.)"
  },
  "renewAt": {
    "type": "integer",
    "description": "Unix timestamp (seconds) when the resource should be renewed",
    "optional": true
  },
  "private": {
    "type": "object",
    "additionalProperties": true,
    "description": "Private data passed to renew() and close() methods (not visible to user)",
    "optional": true
  }
}
```

**Example Request**:

```json
{ "jsonrpc": "2.0", "id": 1, "method": "open", "params": { "props": { "vault_path": "secret/data/prod" } } }
```

**Example Response**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "result": { "username": "admin", "password": "secret123" },
    "renewAt": 1738252800,
    "private": { "lease_id": "abc123" }
  }
}
```

#### renew (optional)

Renews an ephemeral resource before it expires.

**Error Handling**: If this method is not implemented, return error code `-32601` (Method not found). The provider will not call it again.

**Parameters**:

```json
{
  "private": {
    "type": "object",
    "additionalProperties": true,
    "description": "Private data from open() or previous renew()"
  }
}
```

**Result**:

```json
{
  "renewAt": {
    "type": "integer",
    "description": "Unix timestamp when the resource should be renewed again",
    "optional": true
  },
  "private": {
    "type": "object",
    "additionalProperties": true,
    "description": "Updated private data for next renew() or close()",
    "optional": true
  }
}
```

**Example Request**:

```json
{ "jsonrpc": "2.0", "id": 1, "method": "renew", "params": { "private": { "lease_id": "abc123" } } }
```

**Example Response**:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { "renewAt": 1738256400, "private": { "lease_id": "abc123", "renew_count": 1 } }
}
```

**Example Response (not implemented)**:

```json
{ "jsonrpc": "2.0", "id": 1, "error": { "code": -32601, "message": "Method not found" } }
```

#### close (optional)

Closes an ephemeral resource and performs cleanup.

**Error Handling**: If this method is not implemented, return error code `-32601` (Method not found). The provider will not call it.

**Parameters**:

```json
{
  "private": {
    "type": "object",
    "additionalProperties": true,
    "description": "Private data from open() or renew()"
  }
}
```

**Result**: No response body (returns `null`)

**Example Request**:

```json
{ "jsonrpc": "2.0", "id": 1, "method": "close", "params": { "private": { "lease_id": "abc123" } } }
```

**Example Response**:

```json
{ "jsonrpc": "2.0", "id": 1, "result": null }
```

**Example Response (not implemented)**:

```json
{ "jsonrpc": "2.0", "id": 1, "error": { "code": -32601, "message": "Method not found" } }
```

---

## Implementation Notes

### Using Base Classes (Recommended)

Instead of implementing the JSON-RPC protocol directly, extend the provided base classes:

```typescript
import { DenoBridgeResource } from "@brad-jones/terraform-provider-denobridge";

export default class MyResource extends DenoBridgeResource<Props, State, string> {
  async create(props: Props): Promise<{ id: string; state: State }> {
    // Implementation
  }

  async read(id: string, props: Props): Promise<{ props?: Props; state?: State; exists?: boolean }> {
    // Implementation
  }

  // ... other methods
}
```

### Entrypoint Generation

The provider automatically generates a TypeScript entrypoint that:

1. Imports your class from the configured script path
2. Creates a JSocket instance connected to stdin/stdout
3. Registers your class methods as JSON-RPC handlers
4. Handles method name translation (e.g., `read` → `"read"`)
5. Manages optional methods (modifyPlan, renew, close)

### Method Name Translation

TypeScript method names are converted to camelCase for JSON-RPC:

- TypeScript: `read()` → RPC: `"read"`
- TypeScript: `modifyPlan()` → RPC: `"modifyPlan"`
- TypeScript: `invokeProgress()` → RPC: `"invokeProgress"`

### Resource ID Flexibility

Resource IDs can be:

- **Strings**: `"abc123"`, `"/path/to/file"`
- **Numbers**: `42`, `12345`
- **Objects**: `{"project": "my-project", "region": "us-central1", "name": "my-instance"}`

Use objects for composite IDs in resources that require multiple identifiers.

### State vs Props

- **Props**: User-configurable input values from Terraform configuration
- **State**: Computed read-only values (timestamps, checksums, API-assigned IDs, internal tracking data)

Props can be updated by the user; state cannot.

---

## Related Documentation

- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [OpenRPC Specification](https://open-rpc.org/)
- [JSR Package Documentation](https://jsr.io/@brad-jones/terraform-provider-denobridge)
- [Deno Permissions Guide](deno-permissions.md)
