import { Elysia, t } from "elysia";
import { db, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq } from "drizzle-orm";
import { nanoid } from "nanoid";
import { DatabaseModels, CommonModels } from "./models";
import { getProjectDb } from "../db/project-db-helpers";
import { projectDatabases } from "../db/project-schema";

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

// Helper function to verify database access
async function verifyDatabaseAccess(
  databaseId: string,
  projectId: string,
  userId: string,
): Promise<{
  database: typeof projectDatabases.$inferSelect;
  project: typeof projects.$inferSelect;
}> {
  // First verify project ownership
  const project = await verifyProjectOwnership(projectId, userId);

  // Get project database
  const projectDb = await getProjectDb(projectId);

  // Find database in project database
  const [database] = await projectDb
    .select()
    .from(projectDatabases)
    .where(eq(projectDatabases.databaseId, databaseId))
    .limit(1);

  if (!database) {
    throw new NotFoundError("Database", databaseId);
  }

  return { database, project };
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
  // Guard for routes with :id param
  .guard(
    {
      params: t.Object({
        id: t.String({
          minLength: 1,
          error: "Database ID is required",
        }),
      }),
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
    },
    (app) =>
      app
        .get(
          "/:id",
          async ({ user, params, query }) => {
            requireAuth(user);
            if (!query.projectId) {
              throw new Error("projectId is required");
            }
            const { database } = await verifyDatabaseAccess(
              params.id,
              query.projectId as string,
              user.id,
            );

            return {
              data: {
                databaseId: database.databaseId,
                name: database.name,
                createdAt: database.createdAt,
                updatedAt: database.updatedAt,
              },
            };
          },
          {
            response: {
              200: t.Object({
                data: DatabaseModels.response,
              }),
            },
          },
        )
        .delete(
          "/:id",
          async ({ user, params, query }) => {
            requireAuth(user);
            if (!query.projectId) {
              throw new Error("projectId is required");
            }
            const { database, project } = await verifyDatabaseAccess(
              params.id,
              query.projectId as string,
              user.id,
            );

            // Get project database and delete
            const projectDb = await getProjectDb(project.id);
            await projectDb
              .delete(projectDatabases)
              .where(eq(projectDatabases.databaseId, params.id));

            return {
              message: "Database deleted successfully",
            };
          },
          {
            response: {
              200: CommonModels.success,
            },
          },
        ),
  )
  .get(
    "/project/:projectId",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      // Get project database
      const projectDb = await getProjectDb(params.projectId);

      const databases = await projectDb
        .select()
        .from(projectDatabases)
        .orderBy(projectDatabases.createdAt);

      return {
        data: databases.map((db) => ({
          databaseId: db.databaseId,
          name: db.name,
          createdAt: db.createdAt,
          updatedAt: db.updatedAt,
        })),
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
  .post(
    "/project/:projectId",
    async ({ user, params, body }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      // Get project database
      const projectDb = await getProjectDb(params.projectId);

      const databaseId = nanoid();

      const [database] = await projectDb
        .insert(projectDatabases)
        .values({
          databaseId: databaseId,
          name: body.name,
        })
        .returning();

      if (!database) {
        throw new Error("Failed to create database");
      }

      return {
        data: {
          databaseId: database.databaseId,
          name: database.name,
          createdAt: database.createdAt,
          updatedAt: database.updatedAt,
        },
      };
    },
    {
      params: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      body: DatabaseModels.create,
      response: {
        200: t.Object({
          data: DatabaseModels.response,
        }),
      },
    },
  );
