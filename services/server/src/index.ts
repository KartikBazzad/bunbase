import { serve } from "bun";
import index from "./index.html";
import app from "./server";
import { logger } from "./lib/logger";
import { initializeLogBuffer } from "./lib/function-log-buffer";
import { flushLogsToStorage } from "./lib/function-log-storage";

// Initialize function log buffer
initializeLogBuffer(flushLogsToStorage);

export const server = serve({
  routes: {
    // Serve index.html for all unmatched routes.
    "/*": index,
    "/api/*": (req: Request) => {
      return app.fetch(req);
    },
  },

  development: process.env.NODE_ENV !== "production" && {
    // Enable browser hot reloading in development
    hmr: true,

    // Echo console logs from the browser to the server
    console: true,
  },
});

logger.info(`🚀 Server running at ${server.url}`);
