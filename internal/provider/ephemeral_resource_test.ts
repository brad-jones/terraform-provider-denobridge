import { Hono } from "hono";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/open", (c) => {
  return c.json({ result: { uuid: crypto.randomUUID() } });
});

export default app satisfies Deno.ServeDefaultExport;
