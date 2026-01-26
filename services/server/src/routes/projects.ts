import { Elysia, t, status } from "elysia";
import { db, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq } from "drizzle-orm";
import { nanoid } from "nanoid";
import { ProjectModels, CommonModels } from "./models";
import {
  initializeProjectDatabase,
  deleteProjectDatabase,
} from "../db/project-db-init";
import { logger } from "../lib/logger";
import {
  logProjectOperation,
  getProjectLogs,
  convertLogsToActivityItems,
} from "../lib/project-logger-utils";

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

export const projectsRoutes = new Elysia({ prefix: "/projects" })
  .resolve(authResolver)
  .model({
    "project.create": ProjectModels.create,
    "project.update": ProjectModels.update,
    "project.params": ProjectModels.params,
    "project.response": ProjectModels.response,
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
      params: ProjectModels.params,
    },
    (app) =>
      app
        .get(
          "/:id",
          async ({ user, params }) => {
            requireAuth(user);
            const project = await verifyProjectOwnership(params.id, user.id);

            return {
              data: {
                id: project.id,
                name: project.name,
                description: project.description,
                ownerId: project.ownerId,
                createdAt: project.createdAt,
                updatedAt: project.updatedAt,
              },
            };
          },
          {
            response: {
              200: t.Object({
                data: ProjectModels.response,
              }),
            },
          },
        )
        .patch(
          "/:id",
          async ({ user, params, body, set }) => {
            requireAuth(user);
            await verifyProjectOwnership(params.id, user.id);

            // Validate at least one field is provided

            const updateData: { name?: string; description?: string } = {};
            if (body.name !== undefined) updateData.name = body.name;
            if (body.description !== undefined)
              updateData.description = body.description;

            const [updated] = await db
              .update(projects)
              .set({
                ...updateData,
                updatedAt: new Date(),
              })
              .where(eq(projects.id, params.id))
              .returning();

            if (!updated) {
              throw new NotFoundError("Project", params.id);
            }

            logProjectOperation(params.id, "update", {
              projectId: updated.id,
              changes: updateData,
            });

            return {
              data: {
                id: updated.id,
                name: updated.name,
                description: updated.description,
                ownerId: updated.ownerId,
                createdAt: updated.createdAt,
                updatedAt: updated.updatedAt,
              },
            };
          },
          {
            body: ProjectModels.update,
            response: {
              200: t.Object({
                data: ProjectModels.response,
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
        .delete(
          "/:id",
          async ({ user, params }) => {
            requireAuth(user);
            await verifyProjectOwnership(params.id, user.id);

            // Delete project database first
            try {
              await deleteProjectDatabase(params.id);
            } catch (error) {
              // Log error but continue with project deletion
              logger.error("Failed to delete project database", error, {
                projectId: params.id,
              });
            }

            // Log deletion before actually deleting
            logProjectOperation(params.id, "delete", {
              projectId: params.id,
            });

            // Delete project record from backend database
            await db.delete(projects).where(eq(projects.id, params.id));

            return {
              message: "Project deleted successfully",
            };
          },
          {
            response: {
              200: CommonModels.success,
            },
          },
        )
        .get(
          "/:id/logs",
          async ({ user, params, query }) => {
            requireAuth(user);
            await verifyProjectOwnership(params.id, user.id);

            const limit = query.limit ? parseInt(query.limit as string) : 50;
            const offset = query.offset
              ? parseInt(query.offset as string)
              : undefined;
            const level = query.level as
              | "debug"
              | "info"
              | "warn"
              | "error"
              | undefined;
            const type = query.type as string | undefined;
            const search = query.search as string | undefined;
            const startDate = query.startDate
              ? new Date(query.startDate as string)
              : undefined;
            const endDate = query.endDate
              ? new Date(query.endDate as string)
              : undefined;

            // Check if we should return full logs or activity items (for backward compatibility)
            const returnFullLogs = query.full === "true";

            const logs = await getProjectLogs(params.id, {
              limit,
              offset,
              level,
              type,
              search,
              startDate,
              endDate,
            });

            if (returnFullLogs) {
              // Return full log records
              return {
                data: logs.map((log) => ({
                  id: log.id.toString(),
                  level: log.level,
                  message: log.message,
                  context: log.context,
                  metadata: log.metadata,
                  correlationId: log.correlationId,
                  service: log.service,
                  type: log.type,
                  timestamp: log.timestamp,
                  projectId: log.projectId,
                })),
                total: logs.length,
                hasMore: logs.length === limit,
              };
            }

            // Return activity items for backward compatibility
            const activities = convertLogsToActivityItems(logs);

            return {
              data: activities,
            };
          },
          {
            query: t.Object({
              limit: t.Optional(t.String()),
              offset: t.Optional(t.String()),
              level: t.Optional(t.String()),
              type: t.Optional(t.String()),
              search: t.Optional(t.String()),
              startDate: t.Optional(t.String()),
              endDate: t.Optional(t.String()),
              full: t.Optional(t.String()),
            }),
            response: {
              200: t.Union([
                t.Object({
                  data: t.Array(
                    t.Object({
                      id: t.String(),
                      title: t.String(),
                      description: t.Optional(t.String()),
                      timestamp: t.Date(),
                      type: t.Optional(
                        t.Union([
                          t.Literal("success"),
                          t.Literal("warning"),
                          t.Literal("error"),
                          t.Literal("info"),
                        ]),
                      ),
                    }),
                  ),
                }),
                t.Object({
                  data: t.Array(
                    t.Object({
                      id: t.String(),
                      level: t.String(),
                      message: t.String(),
                      context: t.Optional(t.Any()),
                      metadata: t.Optional(t.Any()),
                      correlationId: t.Optional(t.String()),
                      service: t.Optional(t.String()),
                      type: t.Optional(t.String()),
                      timestamp: t.Date(),
                      projectId: t.String(),
                    }),
                  ),
                  total: t.Optional(t.Number()),
                  hasMore: t.Optional(t.Boolean()),
                }),
              ]),
            },
          },
        ),
  )
  .get(
    "/",
    async ({ user }) => {
      requireAuth(user);
      const userProjects = await db
        .select()
        .from(projects)
        .where(eq(projects.ownerId, user.id));

      return {
        data: userProjects.map((p) => ({
          id: p.id,
          name: p.name,
          description: p.description,
          ownerId: p.ownerId,
          createdAt: p.createdAt,
          updatedAt: p.updatedAt,
        })),
      };
    },
    {
      response: {
        200: t.Object({
          data: t.Array(ProjectModels.response),
        }),
      },
    },
  )
  .post(
    "/",
    async ({ user, body }) => {
      requireAuth(user);
      const projectId = nanoid();

      // Create project record in backend database
      const [project] = await db
        .insert(projects)
        .values({
          id: projectId,
          name: body.name,
          description: body.description,
          ownerId: user.id,
        })
        .returning();

      if (!project) {
        throw new Error("Failed to create project");
      }

      // Initialize project database
      // If this fails, we should rollback the project creation
      try {
        await initializeProjectDatabase(projectId);
      } catch (error) {
        // Rollback: delete the project record if database creation fails
        await db.delete(projects).where(eq(projects.id, projectId));
        throw new Error(
          `Failed to initialize project database: ${error instanceof Error ? error.message : "Unknown error"}`,
        );
      }

      logProjectOperation(projectId, "create", {
        projectId: project.id,
        projectName: project.name,
        ownerId: user.id,
      });

      return {
        data: {
          id: project.id,
          name: project.name,
          description: project.description,
          ownerId: project.ownerId,
          createdAt: project.createdAt,
          updatedAt: project.updatedAt,
        },
      };
    },
    {
      body: ProjectModels.create,
      response: {
        200: t.Object({
          data: ProjectModels.response,
        }),
      },
    },
  );
