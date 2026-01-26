import { Elysia, t } from "elysia";
import { db, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq } from "drizzle-orm";
import { DatabaseModels, CommonModels } from "./models";

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


export const databasesRoutes = new Elysia({ prefix: "/databases" })
  .resolve(authResolver)
  .model({
    "database.create": DatabaseModels.create,
    "database.params": DatabaseModels.params,
    "database.response": DatabaseModels.response,
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

      // Return single default database info (no actual database table)
      return {
        data: [
          {
            databaseId: "default",
            name: "Default Database",
            projectId: params.projectId,
            createdAt: new Date(),
            updatedAt: new Date(),
          },
        ],
      };
    },
    {
      params: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: t.Array(DatabaseModels.response),
        }),
      },
    },
  )
  // POST route removed - we don't create databases anymore (single database per project)
