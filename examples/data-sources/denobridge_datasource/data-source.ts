import { DatasourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  query: string;
  recordType: "A" | "AAAA" | "ANAME" | "CNAME" | "NS" | "PTR";
}

interface Result {
  ips: string[];

  // Values nested under the "sensitive" key are automatically marked as sensitive
  // in Terraform state, but they are still stored in state and passed to all methods,
  // it just stops them from being displayed in CLI output.
  sensitive: {
    token: string;
  };
}

new DatasourceProvider<Props, Result>({
  async read({ query, recordType }) {
    return {
      ips: await Deno.resolveDns(query, recordType, {
        nameServer: { ipAddr: "1.1.1.1", port: 53 },
      }),
      sensitive: {
        token: "secret-api-token",
      },
    };
  },
});
