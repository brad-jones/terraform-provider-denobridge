import { DenoBridgeDatasource } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  query: string;
  recordType: "A" | "AAAA" | "CNAME" | "MX" | "TXT";
}

export default class DnsLookupDatasource extends DenoBridgeDatasource<Props, string[]> {
  async read(props: Props) {
    const result = await Deno.resolveDns(props.query, props.recordType, {
      nameServer: { ipAddr: "1.1.1.1", port: 53 },
    });
    return result as string[];
  }
}
