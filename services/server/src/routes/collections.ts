import { Elysia, t } from "elysia";
import { db, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq, and, isNull } from "drizzle-orm";
import { nanoid } from "nanoid";
import { CollectionModels, CommonModels } from "./models";
import { getProjectDb } from "../db/project-db-helpers";
import {
  projectDatabases,
  projectCollections,
  projectDocuments,
} from "../db/project-schema";

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

// Helper function to generate collection path
function generateCollectionPath(
  name: string,
  parentPath?: string | null,
  parentDocumentId?: string | null,
): string {
  if (parentPath && parentDocumentId) {
    // Subcollection: parentPath/documentId/collectionName
    return `${parentPath}/${parentDocumentId}/${name}`;
  }
  // Root collection: just the name
  return name;
}

export const collectionsRoutes = new Elysia({
  prefix: "/databases/:id/collections",
})
  .resolve(authResolver)
  .model({
    "collection.create": CollectionModels.create,
    "collection.update": CollectionModels.update,
    "collection.params": CollectionModels.params,
    "collection.response": CollectionModels.response,
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
  // Guard for routes with :collectionId param
  .guard(
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
        collectionId: t.String({ minLength: 1 }),
      }),
    },
    (app) =>
      app
        .get(
          "/:collectionId",
          async ({ user, params, query }) => {
            requireAuth(user);
            if (!query.projectId) {
              throw new Error("projectId query parameter is required");
            }
            await verifyDatabaseAccess(
              params.id,
              query.projectId as string,
              user.id,
            );

            const projectDb = await getProjectDb(projectId);
            const [collection] = await projectDb
              .select()
              .from(projectCollections)
              .where(
                and(
                  eq(projectCollections.collectionId, params.collectionId),
                  eq(projectCollections.databaseId, params.id),
                ),
              )
              .limit(1);

            if (!collection) {
              throw new NotFoundError("Collection", params.collectionId);
            }

            return {
              data: {
                collectionId: collection.collectionId,
                databaseId: collection.databaseId,
                name: collection.name,
                path: collection.path,
                parentDocumentId: collection.parentDocumentId,
                parentPath: collection.parentPath,
                createdAt: collection.createdAt,
                updatedAt: collection.updatedAt,
              },
            };
          },
          {
            query: t.Object({
              projectId: t.String({ minLength: 1 }),
            }),
            response: {
              200: t.Object({
                data: CollectionModels.response,
              }),
            },
          },
        )
        .patch(
          "/:collectionId",
          async ({ user, params, body, query }) => {
            requireAuth(user);
            if (!query.projectId) {
              throw new Error("projectId query parameter is required");
            }
            await verifyDatabaseAccess(
              params.id,
              query.projectId as string,
              user.id,
            );

            const projectDb = await getProjectDb(query.projectId as string);
            const [collection] = await projectDb
              .select()
              .from(projectCollections)
              .where(
                and(
                  eq(projectCollections.collectionId, params.collectionId),
                  eq(projectCollections.databaseId, params.id),
                ),
              )
              .limit(1);

            if (!collection) {
              throw new NotFoundError("Collection", params.collectionId);
            }

            // Update only if name is provided
            if (body.name) {
              // Generate new path if name changed
              const newPath = generateCollectionPath(
                body.name,
                collection.parentPath,
                collection.parentDocumentId,
              );

              const [updated] = await projectDb
                .update(projectCollections)
                .set({
                  name: body.name,
                  path: newPath,
                  updatedAt: new Date(),
                })
                .where(eq(projectCollections.collectionId, params.collectionId))
                .returning();

              if (!updated) {
                throw new Error("Failed to update collection");
              }

              return {
                data: {
                  collectionId: updated.collectionId,
                  databaseId: updated.databaseId,
                  name: updated.name,
                  path: updated.path,
                  parentDocumentId: updated.parentDocumentId,
                  parentPath: updated.parentPath,
                  createdAt: updated.createdAt,
                  updatedAt: updated.updatedAt,
                },
              };
            }

            return {
              data: {
                collectionId: collection.collectionId,
                databaseId: collection.databaseId,
                name: collection.name,
                path: collection.path,
                parentDocumentId: collection.parentDocumentId,
                parentPath: collection.parentPath,
                createdAt: collection.createdAt,
                updatedAt: collection.updatedAt,
              },
            };
          },
          {
            query: t.Object({
              projectId: t.String({ minLength: 1 }),
            }),
            body: CollectionModels.update,
            response: {
              200: t.Object({
                data: CollectionModels.response,
              }),
            },
          },
        )
        .delete(
          "/:collectionId",
          async ({ user, params, query }) => {
            requireAuth(user);
            if (!query.projectId) {
              throw new Error("projectId query parameter is required");
            }
            await verifyDatabaseAccess(
              params.id,
              query.projectId as string,
              user.id,
            );

            const projectDb = await getProjectDb(query.projectId as string);
            const [collection] = await projectDb
              .select()
              .from(projectCollections)
              .where(
                and(
                  eq(projectCollections.collectionId, params.collectionId),
                  eq(projectCollections.databaseId, params.id),
                ),
              )
              .limit(1);

            if (!collection) {
              throw new NotFoundError("Collection", params.collectionId);
            }

            // Delete collection (cascade will handle documents and subcollections)
            await projectDb
              .delete(projectCollections)
              .where(eq(projectCollections.collectionId, params.collectionId));

            return {
              message: "Collection deleted successfully",
            };
          },
          {
            query: t.Object({
              projectId: t.String({ minLength: 1 }),
            }),
            response: {
              200: CommonModels.success,
            },
          },
        ),
  )
  .get(
    "/by-path",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyDatabaseAccess(params.id, query.projectId as string, user.id);

      if (!query.path) {
        throw new NotFoundError("Collection", "path not provided");
      }

      const projectDb = await getProjectDb(query.projectId as string);
      const [collection] = await projectDb
        .select()
        .from(projectCollections)
        .where(
          and(
            eq(projectCollections.path, query.path),
            eq(projectCollections.databaseId, params.id),
          ),
        )
        .limit(1);

      if (!collection) {
        throw new NotFoundError("Collection", query.path);
      }

      return {
        data: {
          collectionId: collection.collectionId,
          databaseId: collection.databaseId,
          name: collection.name,
          path: collection.path,
          parentDocumentId: collection.parentDocumentId,
          parentPath: collection.parentPath,
          createdAt: collection.createdAt,
          updatedAt: collection.updatedAt,
        },
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: CollectionModels.response,
        }),
      },
    },
  )
  .get(
    "/",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyDatabaseAccess(params.id, query.projectId as string, user.id);

      const projectDb = await getProjectDb(query.projectId as string);

      // If parentPath is provided, get subcollections
      if (query.parentPath) {
        const subcollections = await projectDb
          .select()
          .from(projectCollections)
          .where(
            and(
              eq(projectCollections.databaseId, params.id),
              eq(projectCollections.parentPath, query.parentPath),
            ),
          );

        return {
          data: subcollections.map((col) => ({
            collectionId: col.collectionId,
            databaseId: col.databaseId,
            name: col.name,
            path: col.path,
            parentDocumentId: col.parentDocumentId,
            parentPath: col.parentPath,
            createdAt: col.createdAt,
            updatedAt: col.updatedAt,
          })),
        };
      }

      // Otherwise, get root collections (no parentPath)
      const rootCollections = await projectDb
        .select()
        .from(projectCollections)
        .where(
          and(
            eq(projectCollections.databaseId, params.id),
            isNull(projectCollections.parentPath),
          ),
        );

      return {
        data: rootCollections.map((col) => ({
          collectionId: col.collectionId,
          databaseId: col.databaseId,
          name: col.name,
          path: col.path,
          parentDocumentId: col.parentDocumentId,
          parentPath: col.parentPath,
          createdAt: col.createdAt,
          updatedAt: col.updatedAt,
        })),
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
        parentPath: t.Optional(t.String()),
      }),
      response: {
        200: t.Object({
          data: CollectionModels.listResponse,
        }),
      },
    },
  )
  .post(
    "/",
    async ({ user, params, body, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyDatabaseAccess(params.id, query.projectId as string, user.id);

      const projectDb = await getProjectDb(query.projectId as string);

      const collectionId = nanoid();
      const path = generateCollectionPath(
        body.name,
        body.parentPath || null,
        body.parentDocumentId || null,
      );

      // Check if path already exists
      const [existing] = await projectDb
        .select()
        .from(projectCollections)
        .where(
          and(
            eq(projectCollections.path, path),
            eq(projectCollections.databaseId, params.id),
          ),
        )
        .limit(1);

      if (existing) {
        throw new Error("Collection with this path already exists");
      }

      // If creating a subcollection, verify parent document exists
      if (body.parentDocumentId) {
        const [parentDoc] = await projectDb
          .select()
          .from(projectDocuments)
          .where(eq(projectDocuments.documentId, body.parentDocumentId))
          .limit(1);

        if (!parentDoc) {
          throw new NotFoundError("Parent document", body.parentDocumentId);
        }
      }

      const [collection] = await projectDb
        .insert(projectCollections)
        .values({
          collectionId: collectionId,
          databaseId: params.id,
          name: body.name,
          path: path,
          parentDocumentId: body.parentDocumentId || null,
          parentPath: body.parentPath || null,
        })
        .returning();

      if (!collection) {
        throw new Error("Failed to create collection");
      }

      return {
        data: {
          collectionId: collection.collectionId,
          databaseId: collection.databaseId,
          name: collection.name,
          path: collection.path,
          parentDocumentId: collection.parentDocumentId,
          parentPath: collection.parentPath,
          createdAt: collection.createdAt,
          updatedAt: collection.updatedAt,
        },
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      body: CollectionModels.create,
      response: {
        200: t.Object({
          data: CollectionModels.response,
        }),
      },
    },
  );
