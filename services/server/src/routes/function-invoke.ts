/**
 * Dynamic Function Invocation Route
 * Handles HTTP requests to deployed functions
 */

import { Elysia } from "elysia";
import { apiKeyResolver } from "../middleware/api-key";
import { executeFunction } from "../lib/function-executor";
import { getFunctionById } from "../lib/function-helpers";
import { functionVersions } from "../db/schema";
import { eq } from "drizzle-orm";
import { db } from "../db";
import { db } from "../db";
import { functions } from "../db/schema";
import { eq, and } from "drizzle-orm";

/**
 * Dynamic route handler for function invocation
 * Matches: /functions/:name/*
 */
export const functionInvokeRoutes = new Elysia({ prefix: "/functions" })
  .resolve(apiKeyResolver)
  .onError(({ code, error, set }) => {
    if (code === "NOT_FOUND") {
      set.status = 404;
      return {
        error: {
          message: "Function not found",
          code: "FUNCTION_NOT_FOUND",
        },
      };
    }
    set.status = 500;
    return {
      error: {
        message: error instanceof Error ? error.message : "Internal error",
        code: "INTERNAL_ERROR",
      },
    };
  })
  // Dynamic route: /functions/:name/* - Invoke function by name
  .all(
    "/:name/*",
    async ({ apiKey, params, request, path, query, headers, body }) => {
      const functionName = params.name;

      // Find function by name in project
      const [func] = await db
        .select()
        .from(functions)
        .where(
          and(
            eq(functions.name, functionName),
            eq(functions.projectId, apiKey.projectId),
            eq(functions.status, "deployed"),
          ),
        )
        .limit(1);

      if (!func) {
        throw new Error("Function not found or not deployed");
      }

      // Get the path after function name
      const functionPath = path.replace(`/functions/${functionName}`, "") || "/";

      // Build full URL
      const url = new URL(request.url);
      url.pathname = functionPath;
      if (Object.keys(query).length > 0) {
        for (const [key, value] of Object.entries(query)) {
          url.searchParams.set(key, String(value));
        }
      }

      // Execute function
      const result = await executeFunction(
        func.id,
        apiKey.projectId,
        {
          method: request.method,
          url: url.toString(),
          headers: Object.fromEntries(headers.entries()),
          body: body as any,
        },
      );

      if (!result.success) {
        throw new Error(result.error || "Function execution failed");
      }

      // Return response
      const response = result.response!;
      return new Response(JSON.stringify(response.body), {
        status: response.status,
        headers: response.headers,
      });
    },
  )
  // Also support direct function ID invocation: /functions/:id/invoke/*
  .all(
    "/:id/invoke/*",
    async ({ apiKey, params, request, path, query, headers, body }) => {
      const func = await getFunctionById(params.id, apiKey.projectId);

      if (func.status !== "deployed") {
        throw new Error("Function is not deployed");
      }

      // Get the path after /invoke
      const functionPath = path.replace(`/functions/${params.id}/invoke`, "") || "/";

      // Build full URL
      const url = new URL(request.url);
      url.pathname = functionPath;
      if (Object.keys(query).length > 0) {
        for (const [key, value] of Object.entries(query)) {
          url.searchParams.set(key, String(value));
        }
      }

      // Execute function
      const result = await executeFunction(
        func.id,
        apiKey.projectId,
        {
          method: request.method,
          url: url.toString(),
          headers: Object.fromEntries(headers.entries()),
          body: body as any,
        },
      );

      if (!result.success) {
        throw new Error(result.error || "Function execution failed");
      }

      // Return response
      const response = result.response!;
      return new Response(JSON.stringify(response.body), {
        status: response.status,
        headers: response.headers,
      });
    },
  );
