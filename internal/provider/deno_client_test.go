package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestDenoClient(t *testing.T) {
	// TODO: Update this test to use JSON-RPC instead of HTTP
	t.Skip("Test needs to be updated for JSON-RPC")

	// Enable test mode for direct stdout/stderr logging
	t.Setenv("DENO_TOFU_BRIDGE_TEST_MODE", "true")

	denoBinary, err := exec.LookPath("deno")
	assert.NoError(t, err)

	scriptPath := filepath.Join(t.TempDir(), "test.ts")
	err = os.WriteFile(scriptPath, []byte(`
		import { Hono } from "jsr:@hono/hono";

		const app = new Hono();

		app.get("/health", (c) => {
			return c.json({ status: "ok" });
		});

		app.post("/hello", async (c) => {
			const { name } = await c.req.json();
			return c.json({ message: "Hello " + name });
		});

		export default app satisfies Deno.ServeDefaultExport;
	`), 0644)
	assert.NoError(t, err)

	client := NewDenoClient(denoBinary, scriptPath, "/dev/null", &denoPermissions{All: true}, "datasource")
	err = client.Start(t.Context())
	assert.NoError(t, err)
	defer func() {
		if err := client.Stop(); err != nil {
			t.Errorf("failed to stop client: %v", err)
		}
	}()

	// This section would need to be rewritten for JSON-RPC
	// var response *struct {
	// 	Message string `json:"message"`
	// }
	// err = client.C().Post("/hello").SetBody(map[string]any{"name": "John Smith"}).Do(t.Context()).Into(&response)
	// assert.NoError(t, err)
	// assert.Equal(t, "Hello John Smith", response.Message)
}

func TestLocateDenoConfigFile(t *testing.T) {
	// Clear the cache before running tests
	cachedConfigLookups = make(map[string]string)

	// Get absolute path to test fixtures
	fixturesPath, err := filepath.Abs("../../test_fixtures")
	assert.NoError(t, err)

	// Get absolute path to project root deno.json (since test fixtures are inside the project)
	projectRootConfig, err := filepath.Abs("../../deno.json")
	assert.NoError(t, err)

	tests := []struct {
		name           string
		scriptPath     string
		expectedConfig string // relative to fixtures path, or empty for no config
	}{
		{
			name:           "finds deno.json in same directory",
			scriptPath:     filepath.Join(fixturesPath, "with_deno_json", "script.ts"),
			expectedConfig: filepath.Join(fixturesPath, "with_deno_json", "deno.json"),
		},
		{
			name:           "finds deno.json in parent directory from deep nesting",
			scriptPath:     filepath.Join(fixturesPath, "with_deno_json", "nested", "deep", "script.ts"),
			expectedConfig: filepath.Join(fixturesPath, "with_deno_json", "deno.json"),
		},
		{
			name:           "finds deno.jsonc in same directory",
			scriptPath:     filepath.Join(fixturesPath, "with_deno_jsonc", "script.ts"),
			expectedConfig: filepath.Join(fixturesPath, "with_deno_jsonc", "deno.jsonc"),
		},
		{
			name:           "prefers deno.json over deno.jsonc when both exist",
			scriptPath:     filepath.Join(fixturesPath, "with_deno_json", "script.ts"),
			expectedConfig: filepath.Join(fixturesPath, "with_deno_json", "deno.json"),
		},
		{
			name:           "finds project root config when no local config exists",
			scriptPath:     filepath.Join(fixturesPath, "no_config", "script.ts"),
			expectedConfig: projectRootConfig,
		},
		{
			name:           "finds project root config from nested directory with no local config",
			scriptPath:     filepath.Join(fixturesPath, "no_config", "nested", "script.ts"),
			expectedConfig: projectRootConfig,
		},
		{
			name:           "finds config in parent directory",
			scriptPath:     filepath.Join(fixturesPath, "parent_has_config", "nested", "script.ts"),
			expectedConfig: filepath.Join(fixturesPath, "parent_has_config", "deno.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := locateDenoConfigFile(tt.scriptPath)
			assert.Equal(t, tt.expectedConfig, result)
		})
	}
}

func TestLocateDenoConfigFileWithFileURL(t *testing.T) {
	// Clear the cache before running tests
	cachedConfigLookups = make(map[string]string)

	// Get absolute path to test fixtures
	fixturesPath, err := filepath.Abs("../../test_fixtures")
	assert.NoError(t, err)

	tests := []struct {
		name           string
		scriptPath     string
		expectedConfig string
	}{
		{
			name:           "handles file:// URL on Windows",
			scriptPath:     "file:///" + filepath.ToSlash(filepath.Join(fixturesPath, "with_deno_json", "script.ts")),
			expectedConfig: filepath.Join(fixturesPath, "with_deno_json", "deno.json"),
		},
		{
			name:           "handles file:// URL with nested path",
			scriptPath:     "file:///" + filepath.ToSlash(filepath.Join(fixturesPath, "parent_has_config", "nested", "script.ts")),
			expectedConfig: filepath.Join(fixturesPath, "parent_has_config", "deno.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := locateDenoConfigFile(tt.scriptPath)
			assert.Equal(t, tt.expectedConfig, result)
		})
	}
}

func TestLocateDenoConfigFileCaching(t *testing.T) {
	// Clear the cache before running tests
	cachedConfigLookups = make(map[string]string)

	// Get absolute path to test fixtures
	fixturesPath, err := filepath.Abs("../../test_fixtures")
	assert.NoError(t, err)

	scriptPath := filepath.Join(fixturesPath, "with_deno_json", "script.ts")
	expectedConfig := filepath.Join(fixturesPath, "with_deno_json", "deno.json")

	// First call should search filesystem
	result1 := locateDenoConfigFile(scriptPath)
	assert.Equal(t, expectedConfig, result1)

	// Verify it was cached
	cached, ok := cachedConfigLookups[scriptPath]
	assert.True(t, ok, "result should be cached")
	assert.Equal(t, expectedConfig, cached)

	// Second call should return cached value
	result2 := locateDenoConfigFile(scriptPath)
	assert.Equal(t, expectedConfig, result2)
	assert.Equal(t, result1, result2)
}

func TestLocateDenoConfigFileEdgeCases(t *testing.T) {
	// Clear the cache before running tests
	cachedConfigLookups = make(map[string]string)

	tests := []struct {
		name           string
		scriptPath     string
		expectedConfig string
	}{
		{
			name:           "handles nonexistent path gracefully",
			scriptPath:     filepath.Join("nonexistent", "path", "script.ts"),
			expectedConfig: "",
		},
		{
			name:           "handles empty string",
			scriptPath:     "",
			expectedConfig: "",
		},
		{
			name:           "returns empty string for http:// URLs",
			scriptPath:     "http://example.com/script.ts",
			expectedConfig: "",
		},
		{
			name:           "returns empty string for https:// URLs",
			scriptPath:     "https://example.com/script.ts",
			expectedConfig: "",
		},
		{
			name:           "returns empty string for ftp:// URLs",
			scriptPath:     "ftp://example.com/script.ts",
			expectedConfig: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := locateDenoConfigFile(tt.scriptPath)
			assert.Equal(t, tt.expectedConfig, result)
		})
	}
}
