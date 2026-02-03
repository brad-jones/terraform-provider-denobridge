import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

/**
 * Defines the methods that must be implemented by an action provider.
 * @template TProps - The type of the properties that will be passed to the action.
 */
export type ActionProviderMethods<TProps> = {
  /**
   * Invokes the action with the provided properties.
   * @param props - The properties for the action invocation.
   * @param progressCallback - A callback function to report progress messages during action execution.
   * @returns A promise that resolves when the action completes.
   */
  invoke(props: TProps, progressCallback: (message: string) => Promise<void>): Promise<void>;
};

/**
 * Internal type defining the remote methods available to the JSON-RPC client.
 */
type RemoteMethods = {
  /**
   * Notifies the remote client of progress during action invocation.
   * @param params - Object containing the progress message.
   */
  invokeProgress(params: { message: string }): void;
};

/**
 * Base class for implementing Terraform action providers with JSON-RPC communication.
 * Actions are operations that can be invoked with properties and report progress during execution.
 * @template TProps - The type of the properties that will be passed to the action.
 */
export class ActionProvider<TProps> extends BaseJsonRpcProvider<RemoteMethods> {
  /**
   * Creates a new ActionProvider instance.
   * @param providerMethods - The implementation of the action provider methods.
   */
  constructor(providerMethods: ActionProviderMethods<TProps>) {
    super((client) => ({
      async invoke(params: { props: Record<string, unknown> }) {
        await providerMethods.invoke(
          params.props as TProps,
          (message: string) => client.notify("invokeProgress", { message }),
        );
        return { done: true };
      },
    }));
  }
}

/**
 * Action provider with built-in Zod schema validation for properties.
 * Automatically validates incoming properties against the provided Zod schema before invoking the action.
 * @template TProps - A Zod schema type that defines the shape of the action properties.
 */
export class ZodActionProvider<
  TProps extends z.ZodType,
> extends ActionProvider<z.infer<TProps>> {
  /**
   * Creates a new ZodActionProvider instance with schema validation.
   * @param propsSchema - The Zod schema used to validate action properties.
   * @param providerMethods - The implementation of the action provider methods.
   */
  constructor(
    propsSchema: TProps,
    providerMethods: ActionProviderMethods<z.infer<TProps>>,
  ) {
    super({
      async invoke(props, progressCallback) {
        await providerMethods.invoke(propsSchema.parse(props), progressCallback);
      },
    });
  }
}
