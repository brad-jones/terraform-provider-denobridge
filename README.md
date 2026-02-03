<div align="center">

### 🚀 Build Terraform Providers with TypeScript

_Bridge the infrastructure-as-code world with the TypeScript ecosystem_

[![Built with Deno](https://img.shields.io/badge/Built%20with-Deno-00ADD8?style=flat&logo=deno)](https://deno.com/)
[![JSR Version](https://img.shields.io/jsr/v/%40brad-jones/terraform-provider-denobridge?style=flat&logo=jsr)](https://jsr.io/@brad-jones/terraform-provider-denobridge)
[![Terraform](https://img.shields.io/badge/Terraform-844FBA?style=flat&logo=terraform&logoColor=white)](https://registry.terraform.io/providers/brad-jones/denobridge/latest)
[![OpenTofu](https://img.shields.io/badge/OpenTofu-FFDA18?style=flat&logo=opentofu&logoColor=black)](https://search.opentofu.org/provider/brad-jones/denobridge/latest)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](#)

---

</div>

## 🌟 Overview

**Deno Tofu Bridge** allows you to implement Terraform provider logic in **TypeScript** instead of Go.

Your TypeScript code runs as a Deno process, and the provider communicates with it via a JSON-RPC 2.0 protocol over stdin/stdout. This approach combines the **safety** 🔒 and **simplicity** ✨ of Deno with the **power** ⚡ of Terraform's infrastructure management.

Write provider logic in the language you know and love, with access to the entire npm ecosystem!

## ✨ Features

- 🔄 **Full Resource Lifecycle** - Implement complete CRUD operations for managed resources
- 📊 **Data Sources** - Fetch external data or perform computations
- ⏱️ **Ephemeral Resources** - Create short-lived resources for temporary credentials or tokens
- 🎯 **Actions** - Execute operations like notifications, validations, or external workflows
- 🛡️ **Type Safety** - Write provider logic in TypeScript with full type checking
- 🔐 **Deno Permissions** - Fine-grained security control over what your code can access
- 🔌 **JSON-RPC Protocol** - Simple, well-documented RPC protocol over stdin/stdout
- 📚 **TypeScript Library** - Use the official JSR package for simplified development
- ⚡ **Fast Development** - Rapid prototyping without Go expertise
- 📦 **npm Ecosystem** - Leverage TypeScript's rich package ecosystem

## Quick Start

### Installation

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    denobridge = {
      source = "registry.terraform.io/brad-jones/denobridge"
    }
  }
}

provider "denobridge" {}
```

### Provider Configuration

The provider automatically manages the Deno binary, but you can customize this behavior:

#### Default (Automatic Download)

By default, the provider downloads the latest stable Deno release:

```hcl
provider "denobridge" {
  # No configuration needed - downloads latest Deno automatically
}
```

The provider caches up to 3 Deno versions in your system's temp directory to avoid repeated downloads.

#### Specify Deno Version

To use a specific Deno version, set `deno_version`:

```hcl
provider "denobridge" {
  deno_version = "v2.1.4"  # Use a specific version
}
```

Version format examples:

- `"latest"` - Latest stable GA release (default)
- `"v2.1.4"` - Specific stable version
- `"v2.0.0-rc.1"` - Pre-release version

The provider fetches available versions from the [Deno releases](https://github.com/denoland/deno/releases) on GitHub.

#### Use Custom Deno Binary

To use your own Deno installation, specify the binary path:

```hcl
provider "denobridge" {
  deno_binary_path = "/usr/local/bin/deno"  # Unix/macOS
  # or
  # deno_binary_path = "C:\\Program Files\\deno\\deno.exe"  # Windows
}
```

When `deno_binary_path` is set, the provider skips automatic downloading and uses your specified binary.

**Note**: You can also set the `GITHUB_TOKEN` environment variable to authenticate GitHub API requests, which helps avoid rate limiting when downloading Deno versions.

### Example: File Resource

Create a TypeScript file that manages a text file:

**providers/resource_file.ts**

```typescript
import { ResourceProvider } from "jsr:@brad-jones/terraform-provider-denobridge";

interface Props {
  path: string;
  content: string;
}

interface State {
  mtime: number;
}

new ResourceProvider<Props, State>({
  async create({ path, content }) {
    await Deno.writeTextFile(path, content);
    return {
      id: path,
      state: {
        mtime: (await Deno.stat(path)).mtime!.getTime(),
      },
    };
  },
  async read(id, props) {
    try {
      const content = await Deno.readTextFile(id);
      return {
        props: { path: id, content },
        state: {
          mtime: (await Deno.stat(id)).mtime!.getTime(),
        },
      };
    } catch (e) {
      if (e instanceof Deno.errors.NotFound) {
        return { exists: false };
      }
      throw e;
    }
  },
  async update(id, nextProps, currentProps, currentState) {
    if (nextProps.path !== currentProps.path) {
      throw new Error("Cannot change file path - requires resource replacement");
    }
    await Deno.writeTextFile(id, nextProps.content);
    return { mtime: (await Deno.stat(id)).mtime!.getTime() };
  },
  async delete(id) {
    await Deno.remove(id);
  },
});
```

**main.tf**

```hcl
resource "denobridge_resource" "example_file" {
  path = "${path.module}/providers/resource_file.ts"

  permissions = {
    all = true
  }

  props = {
    path    = "${path.module}/output.txt"
    content = "Hello from Deno!"
  }
}
```

### Example: DNS Data Source

Query DNS records using Deno's built-in DNS resolver:

**providers/datasource_dns_record.ts**

```typescript
import { DatasourceProvider } from "jsr:@brad-jones/terraform-provider-denobridge";

interface Props {
  query: string;
  recordType: "A" | "AAAA" | "ANAME" | "CNAME" | "NS" | "PTR";
}

interface Result {
  ips: string[];
}

new DatasourceProvider<Props, Result>({
  async read({ query, recordType }) {
    return {
      ips: await Deno.resolveDns(query, recordType, {
        nameServer: { ipAddr: "1.1.1.1", port: 53 },
      }),
    };
  },
});
```

**main.tf**

```hcl
data "denobridge_datasource" "mx_record" {
  path = "${path.module}/providers/datasource_dns_record.ts"

  permissions = {
    all = true
  }

  props = {
    query      = "example.com."
    recordType = "MX"
  }
}

output "mail_server" {
  value = data.denobridge_datasource.mx_record.result[0].exchange
}
```

## Resource Types

### `denobridge_resource`

Full CRUD lifecycle management for Terraform managed resources.

**Required Methods:**

- `create` - Create a new resource instance
- `read` - Read the current state
- `update` - Update an existing resource
- `delete` - Delete a resource

**Optional Methods:**

- `modifyPlan` - Modify Terraform plans

**Configuration:**

```hcl
resource "denobridge_resource" "example" {
  path = "${path.module}/providers/my_resource.ts"

  permissions = {
    allow = [
      "read=/tmp",
      "write=/tmp",
      "net=api.example.com",
    ]
  }

  props = {
    # Your resource-specific properties
  }
}
```

### `denobridge_datasource`

Read-only data fetching for Terraform data sources.

**Required Methods:**

- `read` - Fetch data

**Configuration:**

```hcl
data "denobridge_datasource" "example" {
  path = "${path.module}/providers/my_datasource.ts"

  permissions = {
    allow = ["net=api.example.com"]
  }

  props = {
    # Your datasource-specific properties
  }
}
```

### `denobridge_ephemeral_resource`

Short-lived resources that exist only during Terraform operations.

**Required Methods:**

- `open` - Open/create the ephemeral resource

**Optional Methods:**

- `renew` - Renew expiring resources
- `close` - Clean up when no longer needed

**Configuration:**

```hcl
ephemeral "denobridge_ephemeral_resource" "example" {
  path = "${path.module}/providers/my_ephemeral.ts"

  permissions = {
    allow = ["net=auth.example.com"]
  }

  props = {
    # Your ephemeral resource properties
  }
}
```

### `denobridge_action`

Operations that don't manage resources but perform actions.

**Required Methods:**

- `invoke` - Execute the action (with streaming progress notifications)

**Configuration:**

```hcl
resource "denobridge_action" "example" {
  path = "${path.module}/providers/my_action.ts"

  permissions = {
    allow = ["net=webhook.example.com"]
  }

  props = {
    # Your action-specific properties
  }
}
```

## Deno Permissions

The provider supports Deno's [security and permissions model](https://docs.deno.com/runtime/fundamentals/security/#permissions). You can grant all permissions or specify individual allow/deny rules that map directly to Deno CLI flags.

### Basic Usage

```hcl
permissions = {
  all = true  # Maps to --allow-all (use with caution!)
}
```

### Fine-Grained Control

Grant specific permissions using the `allow` list:

```hcl
permissions = {
  allow = [
    "read",                    # --allow-read (all paths)
    "write=/tmp",             # --allow-write=/tmp
    "net=example.com:443",    # --allow-net=example.com:443
    "env=HOME,USER",          # --allow-env=HOME,USER
    "run=curl,whoami",        # --allow-run=curl,whoami
    "sys=hostname,osRelease", # --allow-sys=hostname,osRelease
    "ffi=/path/to/lib.so",    # --allow-ffi=/path/to/lib.so
    "hrtime",                 # --allow-hrtime
  ]
}
```

### Deny Specific Permissions

Deny takes precedence over allow:

```hcl
permissions = {
  allow = ["net"]              # Allow all network access
  deny  = ["net=evil.com"]     # Except evil.com
}
```

### Common Permission Types

- **`read`** - File system read access (e.g., `read`, `read=/tmp,/etc`)
- **`write`** - File system write access (e.g., `write=/tmp`)
- **`net`** - Network access (e.g., `net`, `net=example.com,api.example.com:443`)
- **`env`** - Environment variables (e.g., `env`, `env=HOME,USER`)
- **`run`** - Subprocess execution (e.g., `run=curl,whoami`)
- **`sys`** - System information (e.g., `sys=hostname,osRelease`)
- **`ffi`** - Foreign function interface (e.g., `ffi=/path/to/lib.so`)
- **`hrtime`** - High-resolution time measurement
- **`import`** - Dynamic imports from web (e.g., `import=example.com`)

See [Deno's permission documentation](https://docs.deno.com/runtime/fundamentals/security/#permissions) for complete details.

## API Documentation

Detailed JSON-RPC 2.0 protocol documentation is available in the [docs/guides](docs/guides/) directory:

- [json-rpc-protocol.md](docs/guides/json-rpc-protocol.md) - Complete JSON-RPC protocol specification
- [json-rpc-spec.json](docs/guides/json-rpc-spec.json) - OpenRPC schema for all methods

### TypeScript Library

The recommended way to build providers is using the official TypeScript library available on JSR:

```typescript
import {
  ActionProvider,
  DatasourceProvider,
  EphemeralResourceProvider,
  ResourceProvider,
} from "jsr:@brad-jones/terraform-provider-denobridge";
```

The library handles all JSON-RPC communication, health checks, and protocol details automatically.

## Development

### Prerequisites

- Go 1.25.5 or later
- [Deno](https://deno.com/) for running examples
- [Task](https://taskfile.dev/) for running build tasks

### Building

```bash
# Build the provider
task build

# Run tests
task test

# Run example
task run:example
```

### Project Structure

```
.
├── docs/                   # API specifications and documentation
├── example/                # Example Terraform configurations
│   └── providers/          # Example TypeScript implementations
├── internal/
│   └── provider/           # Provider implementation
├── bin/                    # Built binaries
├── main.go                 # Provider entry point
└── go.mod                  # Go dependencies
```

## How It Works

1. **Provider Initialization**: When Terraform/OpenTofu calls the provider, it reads your TypeScript file path and permissions configuration.

2. **Deno Process Start**: The provider automatically downloads (if needed) and starts Deno, running your TypeScript file as a child process.

3. **JSON-RPC Communication**: The provider communicates with the Deno process via JSON-RPC 2.0 messages over stdin/stdout.

4. **Health Check**: The provider sends a `health` method call to verify the process is responsive.

5. **Lifecycle Management**: The provider invokes appropriate JSON-RPC methods (`create`, `read`, `update`, `delete`, etc.) based on Terraform operations.

6. **Type-Safe Development**: Using the TypeScript library from JSR, you implement type-safe handler functions that the library wires up to the JSON-RPC protocol.

7. **Cleanup**: When Terraform completes, the provider gracefully shuts down the Deno process.

## Use Cases

- **Rapid Prototyping**: Quickly build provider logic without Go expertise
- **API Integration**: Leverage TypeScript's rich ecosystem for API clients
- **Custom Resources**: Manage resources not supported by existing providers
- **Data Transformation**: Use TypeScript for complex data processing
- **Testing**: Write and test provider logic with familiar TypeScript tooling
- **Learning**: Understand Terraform provider concepts without learning Go

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## Related Projects

- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [Deno](https://github.com/denoland/deno)
- [@brad-jones/terraform-provider-denobridge](https://jsr.io/@brad-jones/terraform-provider-denobridge) - Official TypeScript library on JSR

## Acknowledgments

Built with the Terraform Plugin Framework and powered by Deno's secure runtime.
