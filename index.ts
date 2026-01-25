import { serve } from "bun";

const app = serve({
  port: 3000,
  fetch(req) {
    return new Response("Hello World!");
  },
});

console.log(`Listening on http://${app.hostname}:${app.port}`);
