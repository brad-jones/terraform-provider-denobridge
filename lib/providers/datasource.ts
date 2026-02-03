import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

/**
 * Defines the methods that must be implemented by a datasource provider.
 * Datasources are read-only resources that fetch data from external sources.
 * @template TProps - The type of the properties/configuration for the datasource.
 * @template TResult - The type of the data returned by the datasource.
 */
export interface DatasourceProviderMethods<TProps, TResult> {
  /**
   * Reads data from the datasource based on the provided properties.
   * @param props - The properties/configuration for the datasource read operation.
   * @returns A promise that resolves to the data fetched from the datasource.
   */
  read(props: TProps): Promise<TResult>;
}

/**
 * Base class for implementing Terraform datasource providers with JSON-RPC communication.
 * Datasources are read-only and used to fetch data from external sources during Terraform operations.
 * @template TProps - The type of the properties/configuration for the datasource.
 * @template TResult - The type of the data returned by the datasource.
 */
export class DatasourceProvider<TProps, TResult> extends BaseJsonRpcProvider {
  /**
   * Creates a new DatasourceProvider instance.
   * @param providerMethods - The implementation of the datasource provider methods.
   */
  constructor(providerMethods: DatasourceProviderMethods<TProps, TResult>) {
    super(() => ({
      async read(params: { props: unknown }) {
        return { result: await providerMethods.read(params.props as TProps) };
      },
    }));
  }
}

/**
 * Datasource provider with built-in Zod schema validation for both properties and results.
 * Automatically validates incoming properties and outgoing results against the provided Zod schemas.
 * @template TProps - A Zod schema type that defines the shape of the datasource properties.
 * @template TResult - A Zod schema type that defines the shape of the datasource result.
 */
export class ZodDatasourceProvider<TProps extends z.ZodType, TResult extends z.ZodType>
  extends DatasourceProvider<z.infer<TProps>, z.infer<TResult>> {
  /**
   * Creates a new ZodDatasourceProvider instance with schema validation.
   * @param propsSchema - The Zod schema used to validate datasource properties.
   * @param resultSchema - The Zod schema used to validate datasource results.
   * @param providerMethods - The implementation of the datasource provider methods.
   */
  constructor(
    propsSchema: TProps,
    resultSchema: TResult,
    providerMethods: DatasourceProviderMethods<z.infer<TProps>, z.infer<TResult>>,
  ) {
    super({
      async read(props) {
        return resultSchema.parse(
          await providerMethods.read(
            propsSchema.parse(props),
          ),
        );
      },
    });
  }
}
