import { Elysia } from "elysia";
import { cors } from "@elysiajs/cors";
import { authHandler } from "./auth";
import { errorHandler } from "./lib/errors";
import { loggerMiddleware } from "./middleware/logger";
import { projectsRoutes } from "./routes/projects";
import { applicationsRoutes } from "./routes/applications";
import { databasesRoutes } from "./routes/databases";
import { authProvidersRoutes } from "./routes/auth-providers";
import { usersRoutes } from "./routes/users";
import { collectionsRoutes } from "./routes/collections";
import { documentsRoutes } from "./routes/documents";
import { storageApiRoutes } from "./routes/storage";
import { realtimeRoutes } from "./routes/realtime";
import { dbRoutes } from "./routes/db";
import { functionsRoutes } from "./routes/functions";
import { functionInvokeRoutes } from "./routes/function-invoke";

const app = new Elysia({
  prefix: "/api",
})
  .use(
    cors({
      origin: [
        "http://localhost:3000", // Main app
        "http://localhost:3001", // Auth example
        "http://localhost:5173", // Common Vite dev server
        "http://localhost:5174", // Alternative Vite port
      ],
      credentials: true, // Allow cookies for Better Auth
      methods: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
      allowedHeaders: ["Content-Type", "Authorization", "Cookie"],
    }),
  )
  .use(loggerMiddleware)
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
  .use(usersRoutes)
  .use(collectionsRoutes)
  .use(documentsRoutes)
  .use(realtimeRoutes)
  // API Key-based routes (matching requirements) - these handle both API keys and can be used by dashboard
  .use(storageApiRoutes)
  .use(dbRoutes)
  .use(functionsRoutes)
  // Dynamic function invocation routes (must be after functionsRoutes to avoid conflicts)
  .use(functionInvokeRoutes);

export default app;

export type AppType = typeof app;
