import { DenoBridgeEphemeralResource } from "@brad-jones/terraform-provider-denobridge";

interface Result {
  uuid: string;
}

export default class UuidEphemeralResource extends DenoBridgeEphemeralResource<Record<string, never>, Result, never> {
  async open(): Promise<{ result: Result }> {
    return {
      result: {
        uuid: crypto.randomUUID(),
      },
    };
  }
}
