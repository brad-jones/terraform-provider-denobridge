package provider

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DenoClient manages a Deno process and JSON-RPC communication with it.
type DenoClient struct {
	scriptPath     string
	configPath     string
	permissions    *denoPermissions
	denoBinaryPath string
	process        *exec.Cmd
	entrypointPath string
	providerType   string // "datasource", "resource", "action", "ephemeral"
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	ctx            context.Context
}

// NewDenoClient creates a new Deno client for the given script and provider type.
func NewDenoClient(denoBinaryPath, scriptPath, configPath string, permissions *denoPermissions, providerType string) *DenoClient {
	return &DenoClient{
		scriptPath:     scriptPath,
		configPath:     configPath,
		permissions:    permissions,
		denoBinaryPath: denoBinaryPath,
		providerType:   providerType,
	}
}

// Start launches the Deno process with a generated entrypoint script.
func (c *DenoClient) Start(ctx context.Context) error {
	// Store context for logging
	c.ctx = ctx

	// Generate entrypoint script
	entrypointContent, err := c.generateEntrypoint()
	if err != nil {
		return fmt.Errorf("failed to generate entrypoint: %w", err)
	}

	// Create temp file for entrypoint
	// Use a consistent name based on script path and provider type for easier debugging
	scriptHash := sha256.Sum256([]byte(c.scriptPath + c.providerType))
	tempFileName := fmt.Sprintf("denobridge_entrypoint_%s_%s.ts",
		c.providerType,
		hex.EncodeToString(scriptHash[:8]))
	tempFilePath := filepath.Join(os.TempDir(), tempFileName)

	if err := os.WriteFile(tempFilePath, []byte(entrypointContent), 0600); err != nil {
		return fmt.Errorf("failed to write entrypoint script: %w", err)
	}
	c.entrypointPath = tempFilePath

	// Build Deno command arguments
	args := []string{"run", "-q"}

	// Attempt to locate a deno config file if none given
	configPath := c.configPath
	if configPath == "" {
		configPath = locateDenoConfigFile(c.scriptPath)
	}
	if configPath != "" && configPath != "/dev/null" {
		args = append(args, "-c", configPath)
	}

	// Add permissions
	if c.permissions != nil {
		if c.permissions.All {
			args = append(args, "--allow-all")
		} else {
			for _, perm := range c.permissions.Allow {
				args = append(args, fmt.Sprintf("--allow-%s", perm))
			}
			for _, perm := range c.permissions.Deny {
				args = append(args, fmt.Sprintf("--deny-%s", perm))
			}
		}
	}

	args = append(args, tempFilePath)

	// Create command
	c.process = exec.CommandContext(ctx, c.denoBinaryPath, args...)

	// Get stdin/stdout pipes
	stdin, err := c.process.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := c.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := c.process.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Log the full command being executed
	fullCmd := append([]string{c.denoBinaryPath}, args...)
	cmdStr := strings.Join(fullCmd, " ")
	if isTestContext() {
		log.Printf("[DEBUG] Executing Deno command: %s", cmdStr)
	} else {
		tflog.Debug(ctx, fmt.Sprintf("Executing Deno command: %s", cmdStr))
	}

	// Start the process
	if err := c.process.Start(); err != nil {
		return fmt.Errorf("failed to start Deno process: %w", err)
	}

	// Store pipes for socket creation (done by caller)
	c.stdin = stdin
	c.stdout = stdout

	// Start goroutine to pipe stderr to tflog
	go pipeToErrorLog(ctx, stderr, "[deno stderr] ")

	return nil
}

// GetStdin returns the stdin pipe for JSON-RPC communication
func (c *DenoClient) GetStdin() io.WriteCloser {
	return c.stdin
}

// GetStdout returns the stdout pipe for JSON-RPC communication
func (c *DenoClient) GetStdout() io.ReadCloser {
	return c.stdout
}

// Stop terminates the Deno process and cleans up the temporary entrypoint file.
func (c *DenoClient) Stop() error {
	var firstErr error

	// Kill the process
	if c.process != nil && c.process.Process != nil {
		if err := c.process.Process.Kill(); err != nil {
			firstErr = fmt.Errorf("failed to kill Deno process: %w", err)
		}
	}

	// Delete the temporary entrypoint file
	if c.entrypointPath != "" {
		if err := os.Remove(c.entrypointPath); err != nil && !os.IsNotExist(err) {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to remove entrypoint file: %w", err)
			}
		}
	}

	return firstErr
}

// generateEntrypoint generates the TypeScript entrypoint script that wires up
// the user's provider script with JSocket for JSON-RPC communication.
func (c *DenoClient) generateEntrypoint() (string, error) {
	// Resolve script path
	scriptPath := c.scriptPath
	if strings.Contains(scriptPath, "://") {
		// URL-based import (http:// https:// file://)
		parsedURL, err := url.Parse(scriptPath)
		if err != nil {
			return "", fmt.Errorf("failed to parse script URL: %w", err)
		}

		if parsedURL.Scheme == "file" {
			// Convert file:// URL to absolute local path for import
			path := parsedURL.Path
			if len(path) > 2 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			localPath := filepath.FromSlash(path)
			absPath, err := filepath.Abs(localPath)
			if err != nil {
				return "", fmt.Errorf("failed to resolve script path: %w", err)
			}
			scriptPath = absPath
		}
		// For http:// and https:// URLs, use them as-is
	} else {
		// Local file path - convert to absolute
		absPath, err := filepath.Abs(scriptPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve script path: %w", err)
		}
		scriptPath = absPath
	}

	// Determine debug logging based on TF_LOG environment variable
	debugLogging := os.Getenv("TF_LOG") == "DEBUG"

	// Generate entrypoint based on provider type
	switch c.providerType {
	case "datasource":
		return c.generateDatasourceEntrypoint(scriptPath, debugLogging), nil
	case "resource":
		return c.generateResourceEntrypoint(scriptPath, debugLogging), nil
	case "action":
		return c.generateActionEntrypoint(scriptPath, debugLogging), nil
	case "ephemeral":
		return c.generateEphemeralEntrypoint(scriptPath, debugLogging), nil
	default:
		return "", fmt.Errorf("unknown provider type: %s", c.providerType)
	}
}

// generateDatasourceEntrypoint generates the entrypoint for datasource providers
func (c *DenoClient) generateDatasourceEntrypoint(scriptPath string, debugLogging bool) string {
	return fmt.Sprintf(`import { createJSocket } from "jsr:@brad-jones/terraform-provider-denobridge";
import UserDataSource from %s;

await using socket = createJSocket(
  Deno.stdin,
  Deno.stdout,
  { debugLogging: %v }
)(() => ({
  async read(params: { props: Record<string, unknown> }) {
    const instance = new UserDataSource();
    const result = await instance.read(params.props as any);
    return { result };
  }
}));
`, escapeImportPath(scriptPath), debugLogging)
}

// generateResourceEntrypoint generates the entrypoint for resource providers
func (c *DenoClient) generateResourceEntrypoint(scriptPath string, debugLogging bool) string {
	return fmt.Sprintf(`import { createJSocket } from "jsr:@brad-jones/terraform-provider-denobridge";
import UserResource from %s;

await using socket = createJSocket(
  Deno.stdin,
  Deno.stdout,
  { debugLogging: %v }
)(() => {
  const instance = new UserResource();

  return {
    async create(params: { props: Record<string, unknown> }) {
      const result = await instance.create(params.props as any);
      return { id: result.id, state: result.state };
    },

    async read(params: { id: unknown; props: Record<string, unknown> }) {
      const result = await instance.read(params.id as any, params.props as any);
      if (result.exists === false) {
        return { exists: false };
      }
      return { props: result.props, state: result.state, exists: true };
    },

    async update(params: { id: unknown; nextProps: Record<string, unknown>; currentProps: Record<string, unknown>; currentState: Record<string, unknown> }) {
      const state = await instance.update(
        params.id as any,
        params.nextProps as any,
        params.currentProps as any,
        params.currentState as any
      );
      return { state };
    },

    async delete(params: { id: unknown; props: Record<string, unknown>; state: Record<string, unknown> }) {
      await instance.delete(params.id as any, params.props as any, params.state as any);
    },

    async modifyPlan(params: { id: unknown | null; nextProps: Record<string, unknown>; currentProps: Record<string, unknown> | null; currentState: Record<string, unknown> | null }) {
      if (instance.modifyPlan) {
        const result = await instance.modifyPlan(
          params.id as any,
          params.nextProps as any,
          params.currentProps as any,
          params.currentState as any
        );
        return result || {};
      }
      throw new Error("Method not found");
    }
  };
});
`, escapeImportPath(scriptPath), debugLogging)
}

// generateActionEntrypoint generates the entrypoint for action providers
func (c *DenoClient) generateActionEntrypoint(scriptPath string, debugLogging bool) string {
	return fmt.Sprintf(`import { createJSocket } from "jsr:@brad-jones/terraform-provider-denobridge";
import UserAction from %s;

await using socket = createJSocket(
  Deno.stdin,
  Deno.stdout,
  { debugLogging: %v }
)((client) => ({
  async invoke(params: { props: Record<string, unknown> }) {
    const instance = new UserAction();
    const result = await instance.invoke(params.props as any, client);
    return { result };
  }
}));
`, escapeImportPath(scriptPath), debugLogging)
}

// generateEphemeralEntrypoint generates the entrypoint for ephemeral resource providers
func (c *DenoClient) generateEphemeralEntrypoint(scriptPath string, debugLogging bool) string {
	return fmt.Sprintf(`import { createJSocket } from "jsr:@brad-jones/terraform-provider-denobridge";
import UserEphemeralResource from %s;

await using socket = createJSocket(
  Deno.stdin,
  Deno.stdout,
  { debugLogging: %v }
)(() => {
  const instance = new UserEphemeralResource();

  return {
    async open(params: { props: Record<string, unknown> }) {
      const result = await instance.open(params.props as any);
      return result;
    },

    async renew(params: { private: Record<string, unknown> }) {
      if (instance.renew) {
        const result = await instance.renew(params.private as any);
        return result || {};
      }
      throw new Error("Method not found");
    },

    async close(params: { private: Record<string, unknown> }) {
      if (instance.close) {
        await instance.close(params.private as any);
      } else {
        throw new Error("Method not found");
      }
    }
  };
});
`, escapeImportPath(scriptPath), debugLogging)
}

// escapeImportPath converts a file path to a proper TypeScript import string
func escapeImportPath(path string) string {
	// If it's already a URL (http://, https://), return as JSON string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		escaped, _ := json.Marshal(path)
		return string(escaped)
	}

	// For local files on Windows, convert backslashes to forward slashes
	path = filepath.ToSlash(path)

	// For local files, use file:// URL format
	if !strings.HasPrefix(path, "file://") {
		path = "file://" + path
	}

	escaped, _ := json.Marshal(path)
	return string(escaped)
}

// isTestContext returns true if running in a test context.
func isTestContext() bool {
	return os.Getenv("DENO_TOFU_BRIDGE_TEST_MODE") == "true"
}

// pipeToErrorLog reads from a reader and logs each line as error.
func pipeToErrorLog(ctx context.Context, reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	if isTestContext() {
		// In test context, write directly to stderr
		for scanner.Scan() {
			log.Printf("[ERROR] %s%s", prefix, scanner.Text())
		}
	} else {
		// In Terraform context, use tflog
		for scanner.Scan() {
			tflog.Error(ctx, prefix+scanner.Text())
		}
	}
}

// cachedConfigLookups stores config file paths to avoid repeated filesystem lookups.
var cachedConfigLookups = make(map[string]string)

// locateDenoConfigFile searches for a Deno configuration file (deno.json or deno.jsonc)
// starting from the script file's directory and traversing upward through parent
// directories until found or root is reached.
//
// Accepts both regular file paths and file:// URLs.
// Results are cached to avoid repeated filesystem operations for the same file paths.
func locateDenoConfigFile(scriptPath string) string {
	// Convert file URL to path if needed
	if strings.HasPrefix(scriptPath, "file://") {
		parsedURL, err := url.Parse(scriptPath)
		if err == nil && parsedURL.Scheme == "file" {
			// On Windows, url.Parse for file:///C:/path gives Path="/C:/path"
			// We need to remove the leading slash before the drive letter
			path := parsedURL.Path
			if len(path) > 2 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			scriptPath = filepath.FromSlash(path)
		}
	}

	// Check if scriptPath has a protocol scheme other than file://
	// If so, return empty string as remote script loading is not supported
	if strings.Contains(scriptPath, "://") {
		return ""
	}

	// Check cache first
	if cached, ok := cachedConfigLookups[scriptPath]; ok {
		return cached
	}

	// Start from the directory containing the script
	currentDir := filepath.Dir(scriptPath)
	volumeName := filepath.VolumeName(currentDir)

	// Walk up the directory tree
	for {
		// Check for deno.json
		denoJsonPath := filepath.Join(currentDir, "deno.json")
		if _, err := os.Stat(denoJsonPath); err == nil {
			cachedConfigLookups[scriptPath] = denoJsonPath
			return denoJsonPath
		}

		// Check for deno.jsonc
		denoJsoncPath := filepath.Join(currentDir, "deno.jsonc")
		if _, err := os.Stat(denoJsoncPath); err == nil {
			cachedConfigLookups[scriptPath] = denoJsoncPath
			return denoJsoncPath
		}

		// Get parent directory
		parentDir := filepath.Dir(currentDir)

		// Check if we've reached the root
		// On Windows: "C:\" becomes "C:\", on Unix: "/" becomes "/"
		if parentDir == currentDir || parentDir == volumeName || parentDir == string(filepath.Separator) {
			break
		}

		currentDir = parentDir
	}

	// No config file found
	return ""
}
