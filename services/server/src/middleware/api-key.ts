import { Elysia, type Context } from "elysia";
import {
  db,
  applicationApiKeys,
  applications,
  projects,
} from "../db";
import { hashApiKey, validateApiKeyFormat } from "../lib/api-keys";
import { eq, and, isNull } from "drizzle-orm";
import { auth } from "../auth";
import { logger } from "../lib/logger";

// API Key context type
export type ApiKeyContext = {
  apiKeyId: string;
  applicationId: string;
  projectId: string;
};

/**
 * Extract API key from request headers
 * Checks both X-API-Key and Authorization: Bearer <key>
 * Also checks if it's a better-auth session token (starts with session_)
 */
function extractApiKeyFromHeaders(headers: Headers): string | null {
  // Try X-API-Key header first
  const apiKeyHeader = headers.get("x-api-key");
  if (apiKeyHeader) {
    return apiKeyHeader.trim();
  }

  // Try Authorization: Bearer <key>
  const authHeader = headers.get("authorization");
  if (authHeader) {
    const parts = authHeader.split(" ");
    if (parts.length === 2 && parts[0].toLowerCase() === "bearer") {
      const token = parts[1].trim();
      // Skip better-auth session tokens (they start with session_)
      if (!token.startsWith("session_")) {
        return token;
      }
    }
  }

  return null;
}

/**
 * API Key resolver function
 * Used with .resolve() for type-safe context extension
 * Validates API key and resolves application → project → database context
 * Also supports user session authentication for dashboard use
 */
export const apiKeyResolver = async ({ request, status }: Context) => {
  try {
    // First, try to get user session (for dashboard use)
    try {
      const session = await auth.api.getSession({ headers: request.headers });
      if (session?.user) {
        // User is authenticated via session - get their projects
        // For now, we'll need projectId from query/headers for session-based access
        // This is a simplified approach - in production, you might want to get default project
        const projectId =
          request.headers.get("x-project-id") ||
          new URL(request.url).searchParams.get("projectId");

        if (projectId) {
          // Get project for user
          const [project] = await db
            .select()
            .from(projects)
            .where(eq(projects.id, projectId))
            .limit(1);

          if (project && project.ownerId === session.user.id) {
            return {
              apiKey: {
                id: `session-${session.user.id}`,
                applicationId: "",
                projectId: project.id,
              } satisfies ApiKeyContext,
            };
          }
        }
      }
    } catch {
      // Session auth failed, continue to API key auth
    }

    // Extract API key from headers
    const apiKey = extractApiKeyFromHeaders(request.headers);

    if (!apiKey) {
      return status(401, {
        error: {
          message:
            "API key is required. Provide it in X-API-Key header or Authorization: Bearer <key>",
          code: "API_KEY_MISSING",
        },
      });
    }

    // Validate API key format
    if (!validateApiKeyFormat(apiKey)) {
      return status(401, {
        error: {
          message: "Invalid API key format",
          code: "API_KEY_INVALID_FORMAT",
        },
      });
    }

    // Hash the API key
    const keyHash = await hashApiKey(apiKey);

    // Look up API key in database
    const [apiKeyRecord] = await db
      .select()
      .from(applicationApiKeys)
      .where(
        and(
          eq(applicationApiKeys.keyHash, keyHash),
          isNull(applicationApiKeys.revokedAt),
        ),
      )
      .limit(1);

    if (!apiKeyRecord) {
      return status(401, {
        error: {
          message: "Invalid or revoked API key",
          code: "API_KEY_INVALID",
        },
      });
    }

    // Update last used timestamp
    await db
      .update(applicationApiKeys)
      .set({ lastUsedAt: new Date() })
      .where(eq(applicationApiKeys.id, apiKeyRecord.id));

    // Get application to resolve project
    const [application] = await db
      .select()
      .from(applications)
      .where(eq(applications.id, apiKeyRecord.applicationId))
      .limit(1);

    if (!application) {
      return status(500, {
        error: {
          message: "Application not found",
          code: "APPLICATION_NOT_FOUND",
        },
      });
    }

    // Get project
    const [project] = await db
      .select()
      .from(projects)
      .where(eq(projects.id, application.projectId))
      .limit(1);

    if (!project) {
      return status(500, {
        error: {
          message: "Project not found",
          code: "PROJECT_NOT_FOUND",
        },
      });
    }

    return {
      apiKey: {
        id: apiKeyRecord.id,
        applicationId: application.id,
        projectId: project.id,
      } satisfies ApiKeyContext,
    };
  } catch (error) {
    logger.error("API key resolution error", error);
    return status(500, {
      error: {
        message: "Failed to validate API key",
        code: "API_KEY_VALIDATION_ERROR",
      },
    });
  }
};

/**
 * API Key middleware plugin
 * Uses apiKeyResolver internally
 */
export const apiKeyMiddleware = new Elysia({ name: "apiKey" }).resolve(
  apiKeyResolver,
);
