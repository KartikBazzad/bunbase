import { Elysia } from "elysia";
import { cors } from "@elysiajs/cors";
import { authHandler } from "./auth";
import { errorHandler } from "./lib/errors";
import { projectsRoutes } from "./routes/projects";
import { applicationsRoutes } from "./routes/applications";
import { databasesRoutes } from "./routes/databases";
import { authProvidersRoutes } from "./routes/auth-providers";
import { collectionsRoutes } from "./routes/collections";
import { documentsRoutes } from "./routes/documents";
import { storageRoutes } from "./routes/storage";
import { realtimeRoutes } from "./routes/realtime";

const app = new Elysia({
  prefix: "/api",
})
  .use(cors())
  .use(errorHandler)
  // Mount better-auth handler
  .all("/auth/*", async ({ request }) => {
    return authHandler(request);
  })
  // Health check endpoint (no auth required)
  .get("/health", () => {
    return { status: "ok", timestamp: new Date().toISOString() };
  })
  // Dashboard routes (require authentication)
  .use(projectsRoutes)
  .use(applicationsRoutes)
  .use(databasesRoutes)
  .use(authProvidersRoutes)
  .use(collectionsRoutes)
  .use(documentsRoutes)
  .use(storageRoutes)
  .use(realtimeRoutes);

export default app;

export type AppType = typeof app;
