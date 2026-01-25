import { Elysia } from "elysia";

const app = new Elysia({
  prefix: "/api",
}).get("/hello", () => {
  return "Hello World";
});

export default app;

export type AppType = typeof app;
