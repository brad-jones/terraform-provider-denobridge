import { ZodDatasourceProvider } from "@brad-jones/terraform-provider-denobridge";
import { z } from "@zod/zod";

const propsSchema = z.object({
  value: z.string(),
});

const resultSchema = z.object({
  hashedValue: z.string(),
  sensitive: z.object({
    secret: z.string(),
  }),
});

new ZodDatasourceProvider(propsSchema, resultSchema, {
  async read({ value }) {
    const encoder = new TextEncoder();
    const data = encoder.encode(value);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashedValue = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
    return {
      hashedValue,
      sensitive: {
        secret: "datasource-secret",
      },
    };
  },
});
