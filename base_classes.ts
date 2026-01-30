/**
 * Abstract base class for Terraform data sources.
 *
 * Extend this class to create a Terraform data source that fetches external data.
 * Data sources are read-only and always return fresh data on each read.
 *
 * @template TProps - The configuration properties for the data source
 * @template TResult - The computed result data that will be exposed to Terraform
 *
 * @example
 * ```ts
 * import { DenoBridgeDatasource } from "@brad-jones/terraform-provider-denobridge";
 *
 * interface MyProps {
 *   query: string;
 *   recordType: string;
 * }
 *
 * type MyResult = Deno.CaaRecord[];
 *
 * export default class MyDataSource extends DenoBridgeDatasource<MyProps, MyResult> {
 *   override async read({ query, recordType }: MyProps): Promise<MyResult> {
 *     return await Deno.resolveDns(query, recordType, {
 *       nameServer: { ipAddr: "1.1.1.1", port: 53 }
 *     });
 *   }
 * }
 * ```
 */
export abstract class DenoBridgeDatasource<TProps, TResult> {
  /**
   * Read data from the external source.
   *
   * This method is called whenever Terraform needs to fetch data from the data source.
   * It should query the external system and return the computed result.
   *
   * @param props - The configuration properties provided in the Terraform configuration
   * @returns The computed result data to be stored in Terraform state
   * @throws Error if the data cannot be read or the props are invalid
   */
  abstract read(props: TProps): Promise<TResult>;
}

/**
 * Abstract base class for Terraform managed resources.
 *
 * Extend this class to create a Terraform resource that manages infrastructure.
 * Resources support full CRUD operations and can maintain state across operations.
 *
 * @template TProps - The configuration properties for the resource
 * @template TState - Additional computed state data (separate from props)
 * @template TID - The resource identifier type (defaults to string)
 *
 * @example
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
 * export default class FileResource extends DenoBridgeResource<FileProps, FileState> {
 *   override async create(props: FileProps) {
 *     await Deno.writeTextFile(props.path, props.content);
 *     const stat = await Deno.stat(props.path);
 *     return {
 *       id: props.path,
 *       state: { mtime: stat.mtime.getTime() }
 *     };
 *   }
 *
 *   override async read(id: string, props: FileProps) {
 *     try {
 *       const [content, stat] = await Promise.all([
 *         Deno.readTextFile(id),
 *         Deno.stat(id)
 *       ]);
 *       return {
 *         props: { ...props, content },
 *         state: { mtime: stat.mtime.getTime() },
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
 *   override async update(id: string, nextProps: FileProps, currentProps: FileProps, currentState: FileState) {
 *     await Deno.writeTextFile(nextProps.path, nextProps.content);
 *     const stat = await Deno.stat(nextProps.path);
 *     return { mtime: stat.mtime.getTime() };
 *   }
 *
 *   override async delete(id: string) {
 *     await Deno.remove(id);
 *   }
 * }
 * ```
 */
export abstract class DenoBridgeResource<TProps, TState, TID = string> {
  /**
   * Create a new resource instance.
   *
   * This method is called when Terraform needs to create a new resource.
   * It should create the resource in the external system and return its ID and initial state.
   *
   * @param props - The configuration properties for the new resource
   * @returns An object containing the resource ID and computed state
   * @throws Error if the resource cannot be created
   */
  abstract create(props: TProps): Promise<{ id: TID; state: TState }>;

  /**
   * Read the current state of an existing resource.
   *
   * This method is called during Terraform refresh to check if the resource still exists
   * and to fetch its current state. If the resource doesn't exist, return `{ exists: false }`.
   *
   * @param id - The resource identifier
   * @param props - The current configuration properties from Terraform state
   * @returns An object containing the resource props, state, and existence flag
   * @throws Error if there was an error reading the resource (other than not found)
   */
  abstract read(
    id: TID,
    props: TProps,
  ): Promise<{ props: TProps; state: TState; exists: true } | { exists: false }>;

  /**
   * Update an existing resource.
   *
   * This method is called when Terraform detects that the resource configuration has changed
   * and the resource can be updated in-place (no replacement needed).
   *
   * @param id - The resource identifier
   * @param nextProps - The new configuration properties
   * @param currentProps - The current configuration properties
   * @param currentState - The current computed state
   * @returns The updated computed state
   * @throws Error if the resource cannot be updated
   */
  abstract update(id: TID, nextProps: TProps, currentProps: TProps, currentState: TState): Promise<TState>;

  /**
   * Delete a resource.
   *
   * This method is called when Terraform needs to destroy the resource.
   * It should remove the resource from the external system.
   *
   * @param id - The resource identifier
   * @param props - The current configuration properties
   * @param state - The current computed state
   * @throws Error if the resource cannot be deleted
   */
  abstract delete(id: TID, props: TProps, state: TState): Promise<void>;

  /**
   * Modify the plan for a resource change (optional).
   *
   * This method allows you to customize the Terraform plan by:
   * - Modifying proposed property values (e.g., applying defaults)
   * - Forcing resource replacement when required
   * - Adding plan-time diagnostics (warnings/errors)
   *
   * If not implemented, the default plan will be used without modification.
   *
   * @param id - The resource identifier (null for create operations)
   * @param nextProps - The proposed new configuration properties
   * @param currentProps - The current configuration properties (null for create)
   * @param currentState - The current computed state (null for create)
   * @returns An object with optional modifiedProps, requiresReplacement flag, and diagnostics
   *
   * @example
   * ```ts
   * override async modifyPlan(id, nextProps, currentProps, currentState) {
   *   // Apply default value
   *   const modifiedProps = { ...nextProps };
   *   if (!modifiedProps.retryCount) {
   *     modifiedProps.retryCount = 3;
   *   }
   *
   *   // Force replacement if critical property changed
   *   const requiresReplacement = currentProps?.region !== nextProps.region;
   *
   *   return {
   *     modifiedProps,
   *     requiresReplacement,
   *     diagnostics: requiresReplacement ? [{
   *       severity: "warning",
   *       summary: "Resource will be replaced",
   *       detail: "Changing the region requires creating a new resource"
   *     }] : undefined
   *   };
   * }
   * ```
   */
  modifyPlan?(
    id: TID | null,
    nextProps: TProps,
    currentProps: TProps | null,
    currentState: TState | null,
  ): Promise<{
    modifiedProps?: TProps;
    requiresReplacement?: boolean;
    diagnostics?: Array<{
      severity: "error" | "warning";
      summary: string;
      detail: string;
    }>;
  }>;
}

/**
 * Abstract base class for Terraform actions.
 *
 * Extend this class to create a Terraform action that performs one-time operations.
 * Actions are similar to data sources but can make changes and report progress.
 *
 * @template TProps - The input properties for the action
 * @template TResult - The result data returned by the action
 *
 * @example
 * ```ts
 * import { DenoBridgeAction } from "@brad-jones/terraform-provider-denobridge";
 * import type { JSONRPCClient } from "@yieldray/json-rpc-ts";
 *
 * interface DeployProps {
 *   destination: string;
 *   files: string[];
 * }
 *
 * interface DeployResult {
 *   deployedCount: number;
 *   success: boolean;
 * }
 *
 * type GoMethods = {
 *   invokeProgress(params: { message: string }): void;
 * };
 *
 * export default class DeployAction extends DenoBridgeAction<DeployProps, DeployResult> {
 *   override async invoke(
 *     props: DeployProps,
 *     client: JSONRPCClient<GoMethods>
 *   ): Promise<DeployResult> {
 *     await client.invokeProgress({ message: "Starting deployment..." });
 *
 *     let deployedCount = 0;
 *     for (const file of props.files) {
 *       await client.invokeProgress({ message: `Deploying ${file}...` });
 *       // ... deployment logic ...
 *       deployedCount++;
 *     }
 *
 *     await client.invokeProgress({ message: "Deployment complete!" });
 *     return { deployedCount, success: true };
 *   }
 * }
 * ```
 */
export abstract class DenoBridgeAction<TProps, TResult> {
  /**
   * Execute the action.
   *
   * This method is called when the action is invoked. It can perform any operation
   * and optionally send progress notifications back to Terraform using the client.
   *
   * @param props - The input properties for the action
   * @param client - JSON-RPC client for sending progress notifications to Terraform
   * @returns The result data from the action execution
   * @throws Error if the action fails
   */
  abstract invoke(
    props: TProps,
    client: JSONRPCClient<{ invokeProgress(params: { message: string }): void }>,
  ): Promise<TResult>;
}

/**
 * Abstract base class for Terraform ephemeral resources.
 *
 * Extend this class to create a Terraform ephemeral resource that provides short-lived
 * credentials or tokens. Ephemeral resources support opening, renewing, and closing,
 * with private data that's never exposed in logs or state files.
 *
 * @template TProps - The configuration properties for the ephemeral resource
 * @template TResult - The ephemeral result data (e.g., credentials) exposed to Terraform
 * @template TPrivate - Private data stored in Terraform state but never logged
 *
 * @example
 * ```ts
 * import { DenoBridgeEphemeralResource } from "@brad-jones/terraform-provider-denobridge";
 *
 * interface TokenProps {
 *   scopes: string[];
 * }
 *
 * interface TokenResult {
 *   token: string;
 *   expiresAt: string;
 * }
 *
 * interface TokenPrivate {
 *   refreshToken: string;
 * }
 *
 * export default class TokenResource extends DenoBridgeEphemeralResource<
 *   TokenProps,
 *   TokenResult,
 *   TokenPrivate
 * > {
 *   override async open(props: TokenProps) {
 *     const response = await fetch("https://auth.example.com/token", {
 *       method: "POST",
 *       body: JSON.stringify({ scopes: props.scopes })
 *     });
 *     const data = await response.json();
 *
 *     return {
 *       result: {
 *         token: data.accessToken,
 *         expiresAt: data.expiresAt
 *       },
 *       renewAt: Date.parse(data.expiresAt) - 60000, // Renew 1min before expiry
 *       private: {
 *         refreshToken: data.refreshToken
 *       }
 *     };
 *   }
 *
 *   override async renew(private: TokenPrivate) {
 *     const response = await fetch("https://auth.example.com/refresh", {
 *       method: "POST",
 *       headers: { "Authorization": `Bearer ${private.refreshToken}` }
 *     });
 *     const data = await response.json();
 *
 *     return {
 *       renewAt: Date.parse(data.expiresAt) - 60000,
 *       private: {
 *         refreshToken: data.newRefreshToken
 *       }
 *     };
 *   }
 *
 *   override async close(private: TokenPrivate) {
 *     await fetch("https://auth.example.com/revoke", {
 *       method: "POST",
 *       headers: { "Authorization": `Bearer ${private.refreshToken}` }
 *     });
 *   }
 * }
 * ```
 */
export abstract class DenoBridgeEphemeralResource<TProps, TResult, TPrivate> {
  /**
   * Open/create a new ephemeral resource.
   *
   * This method is called when Terraform needs to acquire the ephemeral resource
   * (e.g., obtain credentials or create a temporary token).
   *
   * @param props - The configuration properties for the ephemeral resource
   * @returns An object with the result data, optional renewal timestamp, and private data
   * @throws Error if the resource cannot be opened
   */
  abstract open(props: TProps): Promise<{
    result: TResult;
    renewAt?: number;
    private?: TPrivate;
  }>;

  /**
   * Renew the ephemeral resource (optional).
   *
   * This method is called when Terraform needs to refresh the ephemeral resource
   * before it expires. If not implemented, the resource will not support renewal
   * and Terraform will call open() again to get a new resource.
   *
   * @param private - The private data from the previous open() or renew() call
   * @returns An object with optional renewal timestamp and updated private data
   * @throws Error if the resource cannot be renewed
   */
  renew?(private: TPrivate): Promise<{
    renewAt?: number;
    private?: TPrivate;
  }>;

  /**
   * Close/cleanup the ephemeral resource (optional).
   *
   * This method is called when Terraform is done with the ephemeral resource
   * and needs to clean it up (e.g., revoke credentials or delete a temporary token).
   * If not implemented, no cleanup will be performed.
   *
   * @param private - The private data from the previous open() or renew() call
   * @throws Error if the resource cannot be closed
   */
  close?(private: TPrivate): Promise<void>;
}

/**
 * Re-export JSONRPCClient type for use in action implementations.
 * This allows users to type their client parameter without depending on @yieldray/json-rpc-ts directly.
 */
export type { JSONRPCClient } from "@yieldray/json-rpc-ts";
