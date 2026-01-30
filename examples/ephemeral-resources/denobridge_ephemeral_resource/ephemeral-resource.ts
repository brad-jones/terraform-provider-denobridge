import { DenoBridgeEphemeralResource } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  type: "v4";
}

interface Result {
  uuid: string;
}

export default class UuidEphemeralResource extends DenoBridgeEphemeralResource<Props, Result, never> {
  async open(props: Props): Promise<{ result: Result; renewAt?: number; private?: never }> {
    if (props.type !== "v4") {
      throw new Error("Unsupported UUID type");
    }

    return {
      result: {
        uuid: crypto.randomUUID(),
      },
    };
  }

  // renew and close methods are optional and not implemented for this simple example
}
