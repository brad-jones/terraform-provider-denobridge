import type { JSONRPCClient, JSONRPCMethod, JSONRPCMethods } from "@yieldray/json-rpc-ts";
import { createJSocket } from "../jsocket.ts";

/**
 * Base class for all JSON-RPC provider implementations in the denobridge Terraform provider.
 * Handles the JSON-RPC communication layer over stdin/stdout and provides common functionality
 * like health checks and graceful shutdown.
 * @template RemoteMethods - The type defining the remote methods available to the JSON-RPC client.
 */
export class BaseJsonRpcProvider<RemoteMethods extends JSONRPCMethods = JSONRPCMethods> {
  /**
   * Creates a new BaseJsonRpcProvider instance and initializes the JSON-RPC server.
   * @param providerMethods - A function that receives a JSON-RPC client and returns an object
   *                          containing the provider's method implementations. The client can be
   *                          used to make calls or send notifications to the remote side.
   */
  constructor(providerMethods: (client: JSONRPCClient<RemoteMethods>) => Record<string, unknown>) {
    console.error(
      "This is a JSON-RPC 2.0 server for the denobridge terraform provider. see: https://github.com/brad-jones/terraform-provider-denobridge",
    );

    let debugLogging = false;
    try {
      debugLogging = Deno.env.get("TF_LOG")?.toLowerCase() === "debug";
    } catch {
      // swallow exception due to no permissions to read env vars
    }

    const socket = createJSocket<RemoteMethods>(Deno.stdin, Deno.stdout, { debugLogging })(
      (client) =>
        wrapMethods({
          ...providerMethods(client),
          health() {
            return { ok: true };
          },
          shutdown() {
            console.error("Shutting down gracefully...");
            socket[Symbol.asyncDispose]();
          },
        }),
    );
  }
}

function wrapMethod<T, U>(fn: JSONRPCMethod<T, U>): JSONRPCMethod<T, U> {
  return async (arg) => {
    try {
      return await fn(arg);
    } catch (e) {
      console.error("uncaught error", e);
      throw e;
    }
  };
}

function wrapMethods(methods: JSONRPCMethods): JSONRPCMethods {
  return Object.fromEntries(
    Object.entries(methods).map(([name, fn]) => [
      name,
      typeof fn === "function" ? wrapMethod(fn.bind(methods)) : fn,
    ]),
  );
}
