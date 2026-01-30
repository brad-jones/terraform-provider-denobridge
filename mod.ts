/**
 * @module
 *
 * Terraform Provider DenoBridge - TypeScript SDK
 *
 * This module provides the TypeScript SDK for building Terraform providers using Deno.
 * It includes abstract base classes for all provider types (data sources, resources,
 * actions, and ephemeral resources) as well as the JSocket library for JSON-RPC
 * communication over stdio.
 *
 * ## Quick Start
 *
 * ### Data Source
 *
 * ```ts
 * import { DenoBridgeDatasource } from "@brad-jones/terraform-provider-denobridge";
 *
 * interface MyProps {
 *   query: string;
 * }
 *
 * export default class extends DenoBridgeDatasource<MyProps, string[]> {
 *   override async read({ query }: MyProps) {
 *     // Fetch and return data
 *     return ["result1", "result2"];
 *   }
 * }
 * ```
 *
 * ### Resource
 *
 * ```ts
 * import { DenoBridgeResource } from "@brad-jones/terraform-provider-denobridge";
 *
 * interface FileProps {
 *   path: string;
 *   content: string;
 * }
 *
 * interface FileState {
 *   mtime: number;
 * }
 *
 * export default class extends DenoBridgeResource<FileProps, FileState> {
 *   override async create(props: FileProps) {
 *     await Deno.writeTextFile(props.path, props.content);
 *     const stat = await Deno.stat(props.path);
 *     return { id: props.path, state: { mtime: stat.mtime!.getTime() } };
 *   }
 *
 *   override async read(id: string, props: FileProps) {
 *     try {
 *       const content = await Deno.readTextFile(id);
 *       const stat = await Deno.stat(id);
 *       return {
 *         props: { ...props, content },
 *         state: { mtime: stat.mtime!.getTime() },
 *         exists: true
 *       };
 *     } catch (e) {
 *       if (e instanceof Deno.errors.NotFound) {
 *         return { exists: false };
 *       }
 *       throw e;
 *     }
 *   }
 *
 *   override async update(id: string, nextProps: FileProps) {
 *     await Deno.writeTextFile(nextProps.path, nextProps.content);
 *     const stat = await Deno.stat(nextProps.path);
 *     return { mtime: stat.mtime!.getTime() };
 *   }
 *
 *   override async delete(id: string) {
 *     await Deno.remove(id);
 *   }
 * }
 * ```
 *
 * ### Action
 *
 * ```ts
 * import { DenoBridgeAction, type JSONRPCClient } from "@brad-jones/terraform-provider-denobridge";
 *
 * interface Props {
 *   message: string;
 * }
 *
 * type GoMethods = {
 *   invokeProgress(params: { message: string }): void;
 * };
 *
 * export default class extends DenoBridgeAction<Props, { success: boolean }> {
 *   override async invoke(props: Props, client: JSONRPCClient<GoMethods>) {
 *     await client.invokeProgress({ message: "Starting..." });
 *     // Do work...
 *     await client.invokeProgress({ message: "Complete!" });
 *     return { success: true };
 *   }
 * }
 * ```
 *
 * ### Ephemeral Resource
 *
 * ```ts
 * import { DenoBridgeEphemeralResource } from "@brad-jones/terraform-provider-denobridge";
 *
 * interface Props {
 *   scope: string;
 * }
 *
 * interface Result {
 *   token: string;
 * }
 *
 * interface Private {
 *   refreshToken: string;
 * }
 *
 * export default class extends DenoBridgeEphemeralResource<Props, Result, Private> {
 *   override async open(props: Props) {
 *     // Generate credentials
 *     return {
 *       result: { token: "access-token" },
 *       renewAt: Date.now() + 3600000,
 *       private: { refreshToken: "refresh-token" }
 *     };
 *   }
 *
 *   override async renew(private: Private) {
 *     // Refresh credentials
 *     return {
 *       renewAt: Date.now() + 3600000,
 *       private: { refreshToken: "new-refresh-token" }
 *     };
 *   }
 *
 *   override async close(private: Private) {
 *     // Cleanup/revoke credentials
 *   }
 * }
 * ```
 */

// Export abstract base classes for provider types
export {
  DenoBridgeAction,
  DenoBridgeDatasource,
  DenoBridgeEphemeralResource,
  DenoBridgeResource,
} from "./base_classes.ts";

// Export JSocket for advanced use cases (e.g., custom entrypoint generation)
export { createJSocket, JSocket } from "./jsocket.ts";
export type { CloseableReadable, JSocketOptions, Writeable } from "./jsocket.ts";

// Re-export useful types from dependencies
export type { JSONRPCClient, JSONRPCMethods } from "@yieldray/json-rpc-ts";
