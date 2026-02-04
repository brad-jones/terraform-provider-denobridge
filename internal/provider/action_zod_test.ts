import { ZodActionProvider } from "@brad-jones/terraform-provider-denobridge";
import { z } from "@zod/zod";

const propsSchema = z.object({
  path: z.string(),
  content: z.string(),
});

new ZodActionProvider(propsSchema, {
  async invoke({ path, content }, progressCallback) {
    await progressCallback("validating with Zod schema");
    await progressCallback("about to write file");
    await Deno.writeTextFile(path, content);
    await progressCallback("file written with Zod validation");
  },
});
