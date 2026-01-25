import { Elysia, t } from "elysia";
import { db, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq, and, sql, desc, asc } from "drizzle-orm";
import { nanoid } from "nanoid";
import { DocumentModels, CommonModels } from "./models";
import { getProjectDb } from "../db/project-db-helpers";
import {
  projectDatabases,
  projectCollections,
  projectDocuments,
} from "../db/project-schema";

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

// Helper function to generate document path
function generateDocumentPath(
  collectionPath: string,
  documentId: string,
): string {
  return `${collectionPath}/${documentId}`;
}

// Helper function to build query filters
// Note: Complex filtering is done in-memory after fetching for simplicity
// For production, consider using PostgreSQL JSONB operators directly
function buildFilterQuery(filter: Record<string, any>) {
  // Return filter object to be applied in-memory
  // This is a simplified approach - for better performance, use raw SQL with JSONB operators
  return filter;
}

// Note: Sorting is now done in-memory for simplicity
// For better performance with large datasets, use PostgreSQL JSONB operators

export const documentsRoutes = new Elysia({ prefix: "/databases/:id" })
  .resolve(authResolver)
  .model({
    "document.create": DocumentModels.create,
    "document.update": DocumentModels.update,
    "document.patch": DocumentModels.patch,
    "document.query": DocumentModels.query,
    "document.params": DocumentModels.params,
    "document.response": DocumentModels.response,
    "document.listResponse": DocumentModels.listResponse,
    "document.batch": DocumentModels.batch,
    "document.batchResponse": DocumentModels.batchResponse,
    "document.atomic": DocumentModels.atomic,
    "document.atomicResponse": DocumentModels.atomicResponse,
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
  // Get document by path
  .get(
    "/documents/by-path",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyDatabaseAccess(params.id, query.projectId as string, user.id);

      if (!query.path) {
        throw new NotFoundError("Document", "path not provided");
      }

      const projectDb = await getProjectDb(query.projectId as string);
      const [document] = await projectDb
        .select({
          document: projectDocuments,
          collection: projectCollections,
        })
        .from(projectDocuments)
        .innerJoin(
          projectCollections,
          eq(projectDocuments.collectionId, projectCollections.collectionId),
        )
        .where(
          and(
            eq(projectDocuments.path, query.path),
            eq(projectCollections.databaseId, params.id),
          ),
        )
        .limit(1);

      if (!document) {
        throw new NotFoundError("Document", query.path);
      }

      return {
        data: {
          documentId: document.document.documentId,
          collectionId: document.document.collectionId,
          path: document.document.path,
          data: document.document.data,
          createdAt: document.document.createdAt,
          updatedAt: document.document.updatedAt,
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
          data: DocumentModels.response,
        }),
      },
    },
  )
  // Query documents by collection path
  .get(
    "/documents",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyDatabaseAccess(params.id, query.projectId as string, user.id);

      if (!query.collectionPath) {
        throw new Error("collectionPath is required");
      }

      const projectDb = await getProjectDb(query.projectId as string);

      // Find collection by path
      const [collection] = await projectDb
        .select()
        .from(projectCollections)
        .where(
          and(
            eq(projectCollections.path, query.collectionPath),
            eq(projectCollections.databaseId, params.id),
          ),
        )
        .limit(1);

      if (!collection) {
        throw new NotFoundError("Collection", query.collectionPath);
      }

      // Build query
      const limit = query.limit || 50;
      const offset = query.offset || 0;

      // Get all documents first (for filtering)
      let allDocs = await projectDb
        .select()
        .from(projectDocuments)
        .where(eq(projectDocuments.collectionId, collection.collectionId));

      // Apply filters in-memory (simplified approach)
      if (query.filter) {
        allDocs = allDocs.filter((doc) => {
          for (const [key, value] of Object.entries(query.filter!)) {
            const docValue = doc.data[key];
            if (
              typeof value === "object" &&
              value !== null &&
              !Array.isArray(value)
            ) {
              if ("$gt" in value && Number(docValue) <= Number(value.$gt))
                return false;
              if ("$gte" in value && Number(docValue) < Number(value.$gte))
                return false;
              if ("$lt" in value && Number(docValue) >= Number(value.$lt))
                return false;
              if ("$lte" in value && Number(docValue) > Number(value.$lte))
                return false;
              if ("$ne" in value && docValue === value.$ne) return false;
              if (
                "$in" in value &&
                Array.isArray(value.$in) &&
                !value.$in.includes(docValue)
              )
                return false;
            } else {
              if (docValue !== value) return false;
            }
          }
          return true;
        });
      }

      const total = allDocs.length;

      // Apply sorting
      if (query.sort) {
        const [sortField, sortDir] = Object.entries(query.sort)[0];
        allDocs.sort((a, b) => {
          const aVal = a.data[sortField];
          const bVal = b.data[sortField];
          const comparison = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
          return sortDir === "asc" ? comparison : -comparison;
        });
      } else {
        // Default sort by createdAt desc
        allDocs.sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());
      }

      // Apply pagination
      const docs = allDocs.slice(offset, offset + limit);

      return {
        data: docs.map((doc) => ({
          documentId: doc.documentId,
          collectionId: doc.collectionId,
          path: doc.path,
          data: doc.data,
          createdAt: doc.createdAt,
          updatedAt: doc.updatedAt,
        })),
        total,
        limit,
        offset,
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      query: t.Intersect([
        DocumentModels.query,
        t.Object({
          projectId: t.String({ minLength: 1 }),
        }),
      ]),
      response: {
        200: DocumentModels.listResponse,
      },
    },
  )
  // Guard for collection-specific routes
  .guard(
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
        collectionId: t.String({ minLength: 1 }),
      }),
    },
    (app) =>
      app
        // Get all documents in a collection
        .get(
          "/collections/:collectionId/documents",
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

            // Verify collection belongs to database
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

            const limit = query.limit || 50;
            const offset = query.offset || 0;

            // Get all documents first (for filtering)
            let allDocs = await projectDb
              .select()
              .from(projectDocuments)
              .where(eq(projectDocuments.collectionId, params.collectionId));

            // Apply filters in-memory (simplified approach)
            if (query.filter) {
              allDocs = allDocs.filter((doc) => {
                for (const [key, value] of Object.entries(query.filter!)) {
                  const docValue = doc.data[key];
                  if (
                    typeof value === "object" &&
                    value !== null &&
                    !Array.isArray(value)
                  ) {
                    if ("$gt" in value && Number(docValue) <= Number(value.$gt))
                      return false;
                    if (
                      "$gte" in value &&
                      Number(docValue) < Number(value.$gte)
                    )
                      return false;
                    if ("$lt" in value && Number(docValue) >= Number(value.$lt))
                      return false;
                    if (
                      "$lte" in value &&
                      Number(docValue) > Number(value.$lte)
                    )
                      return false;
                    if ("$ne" in value && docValue === value.$ne) return false;
                    if (
                      "$in" in value &&
                      Array.isArray(value.$in) &&
                      !value.$in.includes(docValue)
                    )
                      return false;
                  } else {
                    if (docValue !== value) return false;
                  }
                }
                return true;
              });
            }

            const total = allDocs.length;

            // Apply sorting
            if (query.sort) {
              const [sortField, sortDir] = Object.entries(query.sort)[0];
              allDocs.sort((a, b) => {
                const aVal = a.data[sortField];
                const bVal = b.data[sortField];
                const comparison = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
                return sortDir === "asc" ? comparison : -comparison;
              });
            } else {
              // Default sort by createdAt desc
              allDocs.sort(
                (a, b) => b.createdAt.getTime() - a.createdAt.getTime(),
              );
            }

            // Apply pagination
            const docs = allDocs.slice(offset, offset + limit);

            return {
              data: docs.map((doc) => ({
                documentId: doc.documentId,
                collectionId: doc.collectionId,
                path: doc.path,
                data: doc.data,
                createdAt: doc.createdAt,
                updatedAt: doc.updatedAt,
              })),
              total,
              limit,
              offset,
            };
          },
          {
            query: t.Intersect([
              DocumentModels.query,
              t.Object({
                projectId: t.String({ minLength: 1 }),
              }),
            ]),
            response: {
              200: DocumentModels.listResponse,
            },
          },
        )
        // Create document
        .post(
          "/collections/:collectionId/documents",
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

            // Verify collection belongs to database
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

            const documentId = nanoid();
            const path = generateDocumentPath(collection.path, documentId);

            const [document] = await projectDb
              .insert(projectDocuments)
              .values({
                documentId: documentId,
                collectionId: params.collectionId,
                path: path,
                data: body.data,
              })
              .returning();

            if (!document) {
              throw new Error("Failed to create document");
            }

            return {
              data: {
                documentId: document.documentId,
                collectionId: document.collectionId,
                path: document.path,
                data: document.data,
                createdAt: document.createdAt,
                updatedAt: document.updatedAt,
              },
            };
          },
          {
            query: t.Object({
              projectId: t.String({ minLength: 1 }),
            }),
            body: DocumentModels.create,
            response: {
              200: t.Object({
                data: DocumentModels.response,
              }),
            },
          },
        )
        // Guard for document-specific routes
        .guard(
          {
            params: t.Object({
              id: t.String({ minLength: 1 }),
              collectionId: t.String({ minLength: 1 }),
              documentId: t.String({ minLength: 1 }),
            }),
          },
          (docApp) =>
            docApp
              // Get document by ID
              .get(
                "/collections/:collectionId/documents/:documentId",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  const [document] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  if (!document) {
                    throw new NotFoundError("Document", params.documentId);
                  }

                  return {
                    data: {
                      documentId: document.documentId,
                      collectionId: document.collectionId,
                      path: document.path,
                      data: document.data,
                      createdAt: document.createdAt,
                      updatedAt: document.updatedAt,
                    },
                  };
                },
                {
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  response: {
                    200: t.Object({
                      data: DocumentModels.response,
                    }),
                  },
                },
              )
              // Full document update
              .put(
                "/collections/:collectionId/documents/:documentId",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  const [document] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  if (!document) {
                    throw new NotFoundError("Document", params.documentId);
                  }

                  const [updated] = await projectDb
                    .update(projectDocuments)
                    .set({
                      data: body.data,
                      updatedAt: new Date(),
                    })
                    .where(eq(projectDocuments.documentId, params.documentId))
                    .returning();

                  if (!updated) {
                    throw new Error("Failed to update document");
                  }

                  return {
                    data: {
                      documentId: updated.documentId,
                      collectionId: updated.collectionId,
                      path: updated.path,
                      data: updated.data,
                      createdAt: updated.createdAt,
                      updatedAt: updated.updatedAt,
                    },
                  };
                },
                {
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  body: DocumentModels.update,
                  response: {
                    200: t.Object({
                      data: DocumentModels.response,
                    }),
                  },
                },
              )
              // Upsert document (create if not exists, update if exists)
              .put(
                "/collections/:collectionId/documents/:documentId/upsert",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  // Check if document exists
                  const [existing] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  let result;
                  if (existing) {
                    // Update existing document
                    const [updated] = await projectDb
                      .update(projectDocuments)
                      .set({
                        data: body.data,
                        updatedAt: new Date(),
                      })
                      .where(eq(projectDocuments.documentId, params.documentId))
                      .returning();

                    result = updated;
                  } else {
                    // Create new document
                    const path = generateDocumentPath(
                      collection.path,
                      params.documentId,
                    );

                    const [created] = await projectDb
                      .insert(projectDocuments)
                      .values({
                        documentId: params.documentId,
                        collectionId: params.collectionId,
                        path,
                        data: body.data,
                        createdAt: new Date(),
                        updatedAt: new Date(),
                      })
                      .returning();

                    result = created;
                  }

                  return {
                    data: {
                      documentId: result.documentId,
                      collectionId: result.collectionId,
                      path: result.path,
                      data: result.data,
                      createdAt: result.createdAt,
                      updatedAt: result.updatedAt,
                    },
                  };
                },
                {
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  body: DocumentModels.update,
                  response: {
                    200: t.Object({
                      data: DocumentModels.response,
                    }),
                  },
                },
              )
              // Partial document update
              .patch(
                "/collections/:collectionId/documents/:documentId",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  const [document] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  if (!document) {
                    throw new NotFoundError("Document", params.documentId);
                  }

                  // Merge existing data with new data
                  const mergedData = { ...document.data, ...body.data };

                  const [updated] = await projectDb
                    .update(projectDocuments)
                    .set({
                      data: mergedData,
                      updatedAt: new Date(),
                    })
                    .where(eq(projectDocuments.documentId, params.documentId))
                    .returning();

                  if (!updated) {
                    throw new Error("Failed to update document");
                  }

                  return {
                    data: {
                      documentId: updated.documentId,
                      collectionId: updated.collectionId,
                      path: updated.path,
                      data: updated.data,
                      createdAt: updated.createdAt,
                      updatedAt: updated.updatedAt,
                    },
                  };
                },
                {
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  body: DocumentModels.patch,
                  response: {
                    200: t.Object({
                      data: DocumentModels.response,
                    }),
                  },
                },
              )
              // Atomic operations (increment, decrement, array operations)
              .post(
                "/collections/:collectionId/documents/:documentId/atomic",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  // Get current document
                  const [document] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  if (!document) {
                    throw new NotFoundError("Document", params.documentId);
                  }

                  // Start with current data
                  const updatedData = { ...document.data };
                  const operationResults: Array<{
                    type: string;
                    field: string;
                    success: boolean;
                  }> = [];

                  // Apply each atomic operation
                  for (const operation of body.operations) {
                    try {
                      if (operation.type === "increment") {
                        const currentValue =
                          (updatedData[operation.field] as number) || 0;
                        updatedData[operation.field] = currentValue + operation.value;
                        operationResults.push({
                          type: "increment",
                          field: operation.field,
                          success: true,
                        });
                      } else if (operation.type === "decrement") {
                        const currentValue =
                          (updatedData[operation.field] as number) || 0;
                        updatedData[operation.field] = currentValue - operation.value;
                        operationResults.push({
                          type: "decrement",
                          field: operation.field,
                          success: true,
                        });
                      } else if (operation.type === "arrayPush") {
                        if (!Array.isArray(updatedData[operation.field])) {
                          updatedData[operation.field] = [];
                        }
                        (updatedData[operation.field] as any[]).push(operation.value);
                        operationResults.push({
                          type: "arrayPush",
                          field: operation.field,
                          success: true,
                        });
                      } else if (operation.type === "arrayRemove") {
                        if (Array.isArray(updatedData[operation.field])) {
                          const arr = updatedData[operation.field] as any[];
                          const index = arr.findIndex(
                            (item) => JSON.stringify(item) === JSON.stringify(operation.value),
                          );
                          if (index !== -1) {
                            arr.splice(index, 1);
                            updatedData[operation.field] = arr;
                            operationResults.push({
                              type: "arrayRemove",
                              field: operation.field,
                              success: true,
                            });
                          } else {
                            operationResults.push({
                              type: "arrayRemove",
                              field: operation.field,
                              success: false,
                            });
                          }
                        } else {
                          operationResults.push({
                            type: "arrayRemove",
                            field: operation.field,
                            success: false,
                          });
                        }
                      } else if (operation.type === "set") {
                        updatedData[operation.field] = operation.value;
                        operationResults.push({
                          type: "set",
                          field: operation.field,
                          success: true,
                        });
                      }
                    } catch (error) {
                      operationResults.push({
                        type: operation.type,
                        field: operation.field,
                        success: false,
                      });
                    }
                  }

                  // Update document with all atomic operations applied
                  const [updated] = await projectDb
                    .update(projectDocuments)
                    .set({
                      data: updatedData,
                      updatedAt: new Date(),
                    })
                    .where(eq(projectDocuments.documentId, params.documentId))
                    .returning();

                  return {
                    data: updated.data,
                    operations: operationResults,
                  };
                },
                {
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  body: DocumentModels.atomic,
                  response: {
                    200: DocumentModels.atomicResponse,
                  },
                },
              )
              // Delete document
              .delete(
                "/collections/:collectionId/documents/:documentId",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  const [document] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  if (!document) {
                    throw new NotFoundError("Document", params.documentId);
                  }

                  await projectDb
                    .delete(projectDocuments)
                    .where(eq(projectDocuments.documentId, params.documentId));

                  return {
                    message: "Document deleted successfully",
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
              )
              // Batch operations
              .post(
                "/collections/:collectionId/documents/batch",
                async ({ user, params, query, body }) => {
                  requireAuth(user);
                  if (!query.projectId) {
                    throw new Error("projectId query parameter is required");
                  }
                  await verifyDatabaseAccess(
                    params.id,
                    query.projectId as string,
                    user.id,
                  );

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  const results: Array<{
                    success: boolean;
                    documentId?: string;
                    error?: string;
                    data?: Record<string, any>;
                  }> = [];
                  let successCount = 0;
                  let errorCount = 0;

                  // Process each operation in the batch
                  for (const operation of body.operations) {
                    try {
                      if (operation.type === "create") {
                        if (!operation.data) {
                          results.push({
                            success: false,
                            error: "Data is required for create operation",
                          });
                          errorCount++;
                          continue;
                        }

                        const documentId = nanoid();
                        const path = generateDocumentPath(
                          collection.path,
                          documentId,
                        );

                        const [newDocument] = await projectDb
                          .insert(projectDocuments)
                          .values({
                            documentId,
                            collectionId: params.collectionId,
                            path,
                            data: operation.data,
                            createdAt: new Date(),
                            updatedAt: new Date(),
                          })
                          .returning();

                        results.push({
                          success: true,
                          documentId: newDocument.documentId,
                          data: newDocument.data,
                        });
                        successCount++;
                      } else if (operation.type === "update" || operation.type === "upsert") {
                        if (!operation.documentId) {
                          results.push({
                            success: false,
                            error: "documentId is required for update/upsert operation",
                          });
                          errorCount++;
                          continue;
                        }

                        if (!operation.data) {
                          results.push({
                            success: false,
                            error: "Data is required for update/upsert operation",
                          });
                          errorCount++;
                          continue;
                        }

                        // Check if document exists
                        const [existing] = await projectDb
                          .select()
                          .from(projectDocuments)
                          .where(
                            and(
                              eq(projectDocuments.documentId, operation.documentId),
                              eq(projectDocuments.collectionId, params.collectionId),
                            ),
                          )
                          .limit(1);

                        if (existing) {
                          // Update existing document
                          const [updated] = await projectDb
                            .update(projectDocuments)
                            .set({
                              data: operation.data,
                              updatedAt: new Date(),
                            })
                            .where(eq(projectDocuments.documentId, operation.documentId))
                            .returning();

                          results.push({
                            success: true,
                            documentId: updated.documentId,
                            data: updated.data,
                          });
                          successCount++;
                        } else if (operation.type === "upsert") {
                          // Create new document for upsert
                          const path = generateDocumentPath(
                            collection.path,
                            operation.documentId,
                          );

                          const [newDocument] = await projectDb
                            .insert(projectDocuments)
                            .values({
                              documentId: operation.documentId,
                              collectionId: params.collectionId,
                              path,
                              data: operation.data,
                              createdAt: new Date(),
                              updatedAt: new Date(),
                            })
                            .returning();

                          results.push({
                            success: true,
                            documentId: newDocument.documentId,
                            data: newDocument.data,
                          });
                          successCount++;
                        } else {
                          results.push({
                            success: false,
                            documentId: operation.documentId,
                            error: "Document not found",
                          });
                          errorCount++;
                        }
                      } else if (operation.type === "delete") {
                        if (!operation.documentId) {
                          results.push({
                            success: false,
                            error: "documentId is required for delete operation",
                          });
                          errorCount++;
                          continue;
                        }

                        await projectDb
                          .delete(projectDocuments)
                          .where(
                            and(
                              eq(projectDocuments.documentId, operation.documentId),
                              eq(projectDocuments.collectionId, params.collectionId),
                            ),
                          );

                        results.push({
                          success: true,
                          documentId: operation.documentId,
                        });
                        successCount++;
                      }
                    } catch (error) {
                      results.push({
                        success: false,
                        documentId: operation.documentId,
                        error: error instanceof Error ? error.message : "Unknown error",
                      });
                      errorCount++;
                    }
                  }

                  return {
                    results,
                    successCount,
                    errorCount,
                  };
                },
                {
                  params: t.Object({
                    id: t.String({ minLength: 1 }),
                    collectionId: t.String({ minLength: 1 }),
                  }),
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  body: DocumentModels.batch,
                  response: {
                    200: DocumentModels.batchResponse,
                  },
                },
              )
              // Get subcollections of a document
              .get(
                "/collections/:collectionId/documents/:documentId/subcollections",
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

                  const projectDb = await getProjectDb(
                    query.projectId as string,
                  );

                  // Verify collection belongs to database
                  const [collection] = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(
                          projectCollections.collectionId,
                          params.collectionId,
                        ),
                        eq(projectCollections.databaseId, params.id),
                      ),
                    )
                    .limit(1);

                  if (!collection) {
                    throw new NotFoundError("Collection", params.collectionId);
                  }

                  const [document] = await projectDb
                    .select()
                    .from(projectDocuments)
                    .where(
                      and(
                        eq(projectDocuments.documentId, params.documentId),
                        eq(projectDocuments.collectionId, params.collectionId),
                      ),
                    )
                    .limit(1);

                  if (!document) {
                    throw new NotFoundError("Document", params.documentId);
                  }

                  // Get subcollections where parentDocumentId matches
                  const subcollections = await projectDb
                    .select()
                    .from(projectCollections)
                    .where(
                      and(
                        eq(projectCollections.databaseId, params.id),
                        eq(
                          projectCollections.parentDocumentId,
                          params.documentId,
                        ),
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
                },
                {
                  query: t.Object({
                    projectId: t.String({ minLength: 1 }),
                  }),
                  response: {
                    200: t.Object({
                      data: t.Array(
                        t.Object({
                          collectionId: t.String(),
                          databaseId: t.String(),
                          name: t.String(),
                          path: t.String(),
                          parentDocumentId: t.Nullable(t.String()),
                          parentPath: t.Nullable(t.String()),
                          createdAt: t.Date(),
                          updatedAt: t.Date(),
                        }),
                      ),
                    }),
                  },
                },
              ),
        ),
  );
