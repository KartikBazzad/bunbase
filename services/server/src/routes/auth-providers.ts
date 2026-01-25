import { Elysia, t } from "elysia";
import { db, projectAuth, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq } from "drizzle-orm";
import { AuthProviderModels, CommonModels } from "./models";

// Helper function to verify project ownership
async function verifyProjectOwnership(
  projectId: string,
  userId: string,
): Promise<typeof projects.$inferSelect> {
  const [project] = await db
    .select()
    .from(projects)
    .where(eq(projects.id, projectId))
    .limit(1);

  if (!project) {
    throw new NotFoundError("Project", projectId);
  }

  if (project.ownerId !== userId) {
    throw new ForbiddenError("You don't have access to this project");
  }

  return project;
}

export const authProvidersRoutes = new Elysia({ prefix: "/auth" })
  .resolve(authResolver)
  .model({
    "authProvider.update": AuthProviderModels.update,
    "authProvider.params": AuthProviderModels.params,
    "authProvider.response": AuthProviderModels.response,
    "common.success": CommonModels.success,
    "common.error": CommonModels.error,
  })
  .onError(({ code, error, set }) => {
    if (code === "VALIDATION") {
      set.status = 422;
      return {
        error: {
          message: error.message,
          code: "VALIDATION_ERROR",
          details: error.all,
        },
      };
    }
    if (error instanceof NotFoundError) {
      set.status = 404;
      return {
        error: {
          message: error.message,
          code: error.code,
        },
      };
    }
    if (error instanceof ForbiddenError) {
      set.status = 403;
      return {
        error: {
          message: error.message,
          code: error.code,
        },
      };
    }
  })
  .get(
    "/project/:projectId",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      // Get or create auth configuration
      let [authConfig] = await db
        .select()
        .from(projectAuth)
        .where(eq(projectAuth.projectId, params.projectId))
        .limit(1);

      if (!authConfig) {
        // Create default auth config
        const [newAuthConfig] = await db
          .insert(projectAuth)
          .values({
            projectId: params.projectId,
            providers: ["email"],
          })
          .returning();
        authConfig = newAuthConfig;
      }

      if (!authConfig) {
        throw new Error("Failed to get auth configuration");
      }

      return {
        data: {
          projectId: authConfig.projectId,
          providers: authConfig.providers as string[],
          createdAt: authConfig.createdAt,
          updatedAt: authConfig.updatedAt,
        },
      };
    },
    {
      params: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: AuthProviderModels.response,
        }),
      },
    },
  )
  .patch(
    "/project/:projectId",
    async ({ user, params, body, set }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      // Validate at least one provider
      if (body.providers.length === 0) {
        set.status = 400;
        return {
          error: {
            message: "At least one provider must be enabled",
            code: "VALIDATION_ERROR",
          },
        };
      }

      // Ensure email is always included
      const providers = Array.from(new Set([...body.providers, "email"]));

      // Update or create auth config
      const [existing] = await db
        .select()
        .from(projectAuth)
        .where(eq(projectAuth.projectId, params.projectId))
        .limit(1);

      let authConfig;
      if (existing) {
        [authConfig] = await db
          .update(projectAuth)
          .set({
            providers: providers as any,
            updatedAt: new Date(),
          })
          .where(eq(projectAuth.projectId, params.projectId))
          .returning();
      } else {
        [authConfig] = await db
          .insert(projectAuth)
          .values({
            projectId: params.projectId,
            providers: providers as any,
          })
          .returning();
      }

      if (!authConfig) {
        throw new Error("Failed to update auth configuration");
      }

      return {
        data: {
          projectId: authConfig.projectId,
          providers: authConfig.providers as string[],
          createdAt: authConfig.createdAt,
          updatedAt: authConfig.updatedAt,
        },
      };
    },
    {
      params: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      body: AuthProviderModels.update,
      response: {
        200: t.Object({
          data: AuthProviderModels.response,
        }),
        400: t.Object({
          error: t.Object({
            message: t.String(),
            code: t.Optional(t.String()),
          }),
        }),
      },
    },
  )
  .post(
    "/project/:projectId/providers/:provider",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      // Get existing auth config
      const [existing] = await db
        .select()
        .from(projectAuth)
        .where(eq(projectAuth.projectId, params.projectId))
        .limit(1);

      const currentProviders = existing
        ? (existing.providers as string[])
        : ["email"];

      // Toggle provider
      const providers = currentProviders.includes(params.provider)
        ? currentProviders.filter((p) => p !== params.provider)
        : [...currentProviders, params.provider];

      // Ensure email is always included
      const finalProviders = Array.from(new Set([...providers, "email"]));

      let authConfig;
      if (existing) {
        [authConfig] = await db
          .update(projectAuth)
          .set({
            providers: finalProviders as any,
            updatedAt: new Date(),
          })
          .where(eq(projectAuth.projectId, params.projectId))
          .returning();
      } else {
        [authConfig] = await db
          .insert(projectAuth)
          .values({
            projectId: params.projectId,
            providers: finalProviders as any,
          })
          .returning();
      }

      if (!authConfig) {
        throw new Error("Failed to update auth configuration");
      }

      return {
        data: {
          projectId: authConfig.projectId,
          providers: authConfig.providers as string[],
          createdAt: authConfig.createdAt,
          updatedAt: authConfig.updatedAt,
        },
        message: `Provider ${params.provider} ${currentProviders.includes(params.provider) ? "disabled" : "enabled"}`,
      };
    },
    {
      params: AuthProviderModels.params,
      response: {
        200: t.Object({
          data: AuthProviderModels.response,
          message: t.String(),
        }),
      },
    },
  );
