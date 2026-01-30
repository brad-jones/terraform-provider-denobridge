import { DenoBridgeDatasource } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  value: string;
}

interface Result {
  hashedValue: string;
}

export default class HashDatasource extends DenoBridgeDatasource<Props, Result> {
  async read(props: Props): Promise<Result> {
    // Hash the value with SHA256
    const encoder = new TextEncoder();
    const data = encoder.encode(props.value);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashedValue = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");

    return { hashedValue };
  }
}
