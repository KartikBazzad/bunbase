import { Elysia, t } from "elysia";
import { db, applications, projects, applicationApiKeys } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq, and, isNull, desc } from "drizzle-orm";
import { nanoid } from "nanoid";
import {
  ApplicationModels,
  ApplicationKeyModels,
  CommonModels,
} from "./models";
import { generateApiKey, hashApiKey, extractKeyParts } from "../lib/api-keys";

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

// Helper function to verify application access
async function verifyApplicationAccess(
  applicationId: string,
  userId: string,
): Promise<{
  application: typeof applications.$inferSelect;
  project: typeof projects.$inferSelect;
}> {
  const [result] = await db
    .select({
      application: applications,
      project: projects,
    })
    .from(applications)
    .innerJoin(projects, eq(applications.projectId, projects.id))
    .where(eq(applications.id, applicationId))
    .limit(1);

  if (!result) {
    throw new NotFoundError("Application", applicationId);
  }

  if (result.project.ownerId !== userId) {
    throw new ForbiddenError("You don't have access to this application");
  }

  return result;
}

export const applicationsRoutes = new Elysia({ prefix: "/applications" })
  .resolve(authResolver)
  .model({
    "application.create": ApplicationModels.create,
    "application.update": ApplicationModels.update,
    "application.params": ApplicationModels.params,
    "application.response": ApplicationModels.response,
    "applicationKey.params": ApplicationKeyModels.params,
    "applicationKey.response": ApplicationKeyModels.response,
    "applicationKey.generateResponse": ApplicationKeyModels.generateResponse,
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
          error: "Application ID is required",
        }),
      }),
    },
    (app) =>
      app
        .get(
          "/:id",
          async ({ user, params }) => {
            requireAuth(user);
            const { application } = await verifyApplicationAccess(
              params.id,
              user.id,
            );

            return {
              data: {
                id: application.id,
                projectId: application.projectId,
                name: application.name,
                description: application.description,
                type: application.type,
                createdAt: application.createdAt,
                updatedAt: application.updatedAt,
              },
            };
          },
          {
            response: {
              200: t.Object({
                data: ApplicationModels.response,
              }),
            },
          },
        )
        .patch(
          "/:id",
          async ({ user, params, body, set }) => {
            requireAuth(user);
            await verifyApplicationAccess(params.id, user.id);

            // Validate at least one field is provided
            if (!body.name && !body.description && !body.type) {
              set.status = 400;
              return {
                error: {
                  message: "At least one field must be provided for update",
                  code: "VALIDATION_ERROR",
                },
              };
            }

            const updateData: {
              name?: string;
              description?: string;
              type?: "web";
            } = {};
            if (body.name !== undefined) updateData.name = body.name;
            if (body.description !== undefined)
              updateData.description = body.description;
            if (body.type !== undefined) updateData.type = body.type;

            const [updated] = await db
              .update(applications)
              .set({
                ...updateData,
                updatedAt: new Date(),
              })
              .where(eq(applications.id, params.id))
              .returning();

            if (!updated) {
              throw new Error("Failed to update application");
            }

            return {
              data: {
                id: updated.id,
                projectId: updated.projectId,
                name: updated.name,
                description: updated.description,
                type: updated.type,
                createdAt: updated.createdAt,
                updatedAt: updated.updatedAt,
              },
            };
          },
          {
            body: ApplicationModels.update,
            response: {
              200: t.Object({
                data: ApplicationModels.response,
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
            await verifyApplicationAccess(params.id, user.id);

            await db.delete(applications).where(eq(applications.id, params.id));

            return {
              message: "Application deleted successfully",
            };
          },
          {
            response: {
              200: CommonModels.success,
            },
          },
        )
        // API Key endpoints
        .post(
          "/:id/keys",
          async ({ user, params }) => {
            requireAuth(user);
            await verifyApplicationAccess(params.id, user.id);

            // Check if there's already an active key
            const [existingKey] = await db
              .select()
              .from(applicationApiKeys)
              .where(
                and(
                  eq(applicationApiKeys.applicationId, params.id),
                  isNull(applicationApiKeys.revokedAt),
                ),
              )
              .limit(1);

            // Revoke existing key if it exists
            if (existingKey) {
              await db
                .update(applicationApiKeys)
                .set({ revokedAt: new Date() })
                .where(eq(applicationApiKeys.id, existingKey.id));
            }

            // Generate new API key
            const fullKey = generateApiKey();
            const keyHash = await hashApiKey(fullKey);
            const { prefix, suffix } = extractKeyParts(fullKey);

            const keyId = nanoid();

            const [apiKey] = await db
              .insert(applicationApiKeys)
              .values({
                id: keyId,
                applicationId: params.id,
                keyHash,
                keyPrefix: prefix,
                keySuffix: suffix,
              })
              .returning();

            if (!apiKey) {
              throw new Error("Failed to create API key");
            }

            // Return full key only once
            return {
              data: {
                id: apiKey.id,
                applicationId: apiKey.applicationId,
                key: fullKey,
                keyPrefix: apiKey.keyPrefix,
                keySuffix: apiKey.keySuffix,
                createdAt: apiKey.createdAt,
              },
            };
          },
          {
            response: {
              200: t.Object({
                data: ApplicationKeyModels.generateResponse,
              }),
            },
          },
        )
        .get(
          "/:id/keys",
          async ({ user, params }) => {
            requireAuth(user);
            await verifyApplicationAccess(params.id, user.id);

            const [apiKey] = await db
              .select()
              .from(applicationApiKeys)
              .where(
                and(
                  eq(applicationApiKeys.applicationId, params.id),
                  isNull(applicationApiKeys.revokedAt),
                ),
              )
              .orderBy(desc(applicationApiKeys.createdAt))
              .limit(1);

            if (!apiKey) {
              return {
                data: null,
              };
            }

            return {
              data: {
                id: apiKey.id,
                applicationId: apiKey.applicationId,
                keyPrefix: apiKey.keyPrefix,
                keySuffix: apiKey.keySuffix,
                createdAt: apiKey.createdAt,
                lastUsedAt: apiKey.lastUsedAt,
                revokedAt: apiKey.revokedAt,
              },
            };
          },
          {
            response: {
              200: t.Object({
                data: t.Nullable(ApplicationKeyModels.response),
              }),
            },
          },
        )
        .delete(
          "/:id/keys",
          async ({ user, params }) => {
            requireAuth(user);
            await verifyApplicationAccess(params.id, user.id);

            const [apiKey] = await db
              .select()
              .from(applicationApiKeys)
              .where(
                and(
                  eq(applicationApiKeys.applicationId, params.id),
                  isNull(applicationApiKeys.revokedAt),
                ),
              )
              .limit(1);

            if (!apiKey) {
              throw new NotFoundError("API Key", "not found");
            }

            await db
              .update(applicationApiKeys)
              .set({ revokedAt: new Date() })
              .where(eq(applicationApiKeys.id, apiKey.id));

            return {
              message: "API key revoked successfully",
            };
          },
          {
            response: {
              200: CommonModels.success,
              404: CommonModels.error,
            },
          },
        ),
  )
  .get(
    "/project/:projectId",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      const apps = await db
        .select()
        .from(applications)
        .where(eq(applications.projectId, params.projectId));

      return {
        data: apps.map((app) => ({
          id: app.id,
          projectId: app.projectId,
          name: app.name,
          description: app.description,
          type: app.type,
          createdAt: app.createdAt,
          updatedAt: app.updatedAt,
        })),
      };
    },
    {
      params: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: t.Array(ApplicationModels.response),
        }),
      },
    },
  )
  .post(
    "/project/:projectId",
    async ({ user, params, body }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.projectId, user.id);

      const applicationId = nanoid();

      const [application] = await db
        .insert(applications)
        .values({
          id: applicationId,
          projectId: params.projectId,
          name: body.name,
          description: body.description,
          type: body.type || "web",
        })
        .returning();

      if (!application) {
        throw new Error("Failed to create application");
      }

      return {
        data: {
          id: application.id,
          projectId: application.projectId,
          name: application.name,
          description: application.description,
          type: application.type,
          createdAt: application.createdAt,
          updatedAt: application.updatedAt,
        },
      };
    },
    {
      params: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      body: ApplicationModels.create,
      response: {
        200: t.Object({
          data: ApplicationModels.response,
        }),
      },
    },
  );
