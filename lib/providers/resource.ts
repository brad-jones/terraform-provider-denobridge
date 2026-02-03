import { JSONRPCMethodNotFoundError } from "@yieldray/json-rpc-ts";
import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

/**
 * Defines the methods that must be implemented by a resource provider.
 * Resources are stateful objects that can be created, read, updated, and deleted (CRUD operations).
 * They maintain both configuration properties and runtime state.
 * @template TProps - The type of the properties/configuration for the resource.
 * @template TState - The type of the runtime state maintained by the resource.
 * @template TID - The type of the resource identifier (defaults to string).
 */
export type ResourceProviderMethods<TProps, TState, TID = string> = {
  /**
   * Creates a new resource with the provided properties.
   * @param props - The properties/configuration for the new resource.
   * @returns A promise that resolves to an object containing the resource ID and initial state.
   */
  create(props: TProps): Promise<{ id: TID; state: TState }>;
  /**
   * Reads an existing resource by its ID and validates it against the expected properties.
   * @param id - The identifier of the resource to read.
   * @param props - The expected properties/configuration of the resource.
   * @returns A promise that resolves to the current properties and state if the resource exists,
   *          or an object with exists: false if the resource no longer exists.
   */
  read(id: TID, props: TProps): Promise<{ props: TProps; state: TState } | { exists: false }>;
  /**
   * Updates an existing resource with new properties.
   * @param id - The identifier of the resource to update.
   * @param nextProps - The new properties/configuration to apply.
   * @param currentProps - The current properties/configuration before the update.
   * @param currentState - The current state before the update.
   * @returns A promise that resolves to the updated state.
   */
  update(
    id: TID,
    nextProps: TProps,
    currentProps: TProps,
    currentState: TState,
  ): Promise<TState>;
  /**
   * Deletes an existing resource.
   * @param id - The identifier of the resource to delete.
   * @param props - The current properties/configuration of the resource.
   * @param state - The current state of the resource.
   * @returns A promise that resolves when the resource is deleted.
   */
  delete(id: TID, props: TProps, state: TState): Promise<void>;
  /**
   * Modifies a Terraform plan before execution. This method is optional and allows customizing
   * the planned changes, adding diagnostics, or indicating that a resource replacement is required.
   * @param id - The identifier of the resource (null for create operations).
   * @param planType - The type of operation being planned: "create", "update", or "delete".
   * @param nextProps - The new properties/configuration after the planned change.
   * @param currentProps - The current properties/configuration (null for create operations).
   * @param currentState - The current state (null for create operations).
   * @returns A promise that resolves to an object with modified properties and/or diagnostics,
   *          a replacement indicator, or undefined to accept the plan as-is.
   */
  modifyPlan?(
    id: TID | null,
    planType: "create" | "update" | "delete",
    nextProps: TProps,
    currentProps: TProps | null,
    currentState: TState | null,
  ): Promise<
    | {
      /** Modified properties to use instead of the originally planned properties. */
      modifiedProps?: TProps;
      /** Diagnostic messages (errors or warnings) to display to the user. */
      diagnostics?: {
        severity: "error" | "warning";
        summary: string;
        detail: string;
        propName?: string;
      }[];
    }
    | {
      /** Whether the resource must be replaced (destroyed and recreated) instead of updated. */
      requiresReplacement: boolean;
    }
    | undefined
  >;
};

/**
 * Base class for implementing Terraform resource providers with JSON-RPC communication.
 * Resources are stateful objects that support full CRUD operations (create, read, update, delete)
 * and maintain both configuration properties and runtime state throughout their lifecycle.
 * @template TProps - The type of the properties/configuration for the resource.
 * @template TState - The type of the runtime state maintained by the resource.
 * @template TID - The type of the resource identifier (defaults to string).
 */
export class ResourceProvider<TProps, TState, TID = string> extends BaseJsonRpcProvider {
  /**
   * Creates a new ResourceProvider instance.
   * @param providerMethods - The implementation of the resource provider methods.
   */
  constructor(providerMethods: ResourceProviderMethods<TProps, TState, TID>) {
    super(() => ({
      async create(params: { props: Record<string, unknown> }) {
        return await providerMethods.create(params.props as TProps);
      },
      async read(params: { id: TID; props: Record<string, unknown> }) {
        return await providerMethods.read(params.id, params.props as TProps);
      },
      async update(
        params: {
          id: TID;
          nextProps: Record<string, unknown>;
          currentProps: Record<string, unknown>;
          currentState: Record<string, unknown>;
        },
      ) {
        const state = await providerMethods.update(
          params.id,
          params.nextProps as TProps,
          params.currentProps as TProps,
          params.currentState as TState,
        );
        return { state };
      },
      async delete(params: { id: TID; props: Record<string, unknown>; state: Record<string, unknown> }) {
        await providerMethods.delete(params.id, params.props as TProps, params.state as TState);
        return { done: true };
      },
      async modifyPlan(
        params: {
          id?: TID;
          planType: "create" | "update" | "delete";
          nextProps: Record<string, unknown>;
          currentProps?: Record<string, unknown>;
          currentState?: Record<string, unknown>;
        },
      ) {
        if (!providerMethods.modifyPlan) {
          throw new JSONRPCMethodNotFoundError();
        }

        const result = await providerMethods.modifyPlan(
          params?.id ?? null,
          params.planType,
          params.nextProps as TProps,
          params.currentProps as TProps ?? null,
          params.currentState as TState ?? null,
        );

        if (result) {
          return result;
        }

        return { noChanges: true };
      },
    }));
  }
}

/**
 * Resource provider with built-in Zod schema validation for properties and state.
 * Automatically validates all data flowing through the resource lifecycle (create, read, update, delete)
 * against the provided Zod schemas.
 * @template TProps - A Zod schema type that defines the shape of the resource properties.
 * @template TState - A Zod schema type that defines the shape of the resource state.
 * @template TID - The type of the resource identifier (defaults to string).
 */
export class ZodResourceProvider<TProps extends z.ZodType, TState extends z.ZodType, TID = string>
  extends ResourceProvider<z.infer<TProps>, z.infer<TState>, TID> {
  /**
   * Creates a new ZodResourceProvider instance with schema validation.
   * @param propsSchema - The Zod schema used to validate resource properties.
   * @param stateSchema - The Zod schema used to validate resource state.
   * @param providerMethods - The implementation of the resource provider methods.
   */
  constructor(
    propsSchema: TProps,
    stateSchema: TState,
    providerMethods: ResourceProviderMethods<z.infer<TProps>, z.infer<TState>, TID>,
  ) {
    const validatedMethods: ResourceProviderMethods<z.infer<TProps>, z.infer<TState>, TID> = {
      async create(props) {
        const { id, state } = await providerMethods.create(propsSchema.parse(props));
        return { id, state: stateSchema.parse(state) };
      },
      async read(id, props) {
        const result = await providerMethods.read(id, propsSchema.parse(props));
        if ("exists" in result) return result;
        return {
          props: propsSchema.parse(result.props),
          state: stateSchema.parse(result.state),
        };
      },
      async update(id, nextProps, currentProps, currentState) {
        return stateSchema.parse(
          await providerMethods.update(
            id,
            propsSchema.parse(nextProps),
            propsSchema.parse(currentProps),
            stateSchema.parse(currentState),
          ),
        );
      },
      async delete(id, props, state) {
        await providerMethods.delete(id, propsSchema.parse(props), stateSchema.parse(state));
      },
    };
    if (providerMethods.modifyPlan) {
      validatedMethods["modifyPlan"] = async (id, planType, nextProps, currentProps, currentState) => {
        const result = await providerMethods.modifyPlan!(
          id,
          planType,
          propsSchema.parse(nextProps),
          propsSchema.parse(currentProps),
          stateSchema.parse(currentState),
        );
        if (!result) return undefined;
        if ("requiresReplacement" in result) return result;
        return { ...result, modifiedProps: propsSchema.parse(result.modifiedProps) };
      };
    }
    super(validatedMethods);
  }
}
