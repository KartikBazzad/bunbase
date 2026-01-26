import { Elysia, t } from "elysia";
import { apiKeyResolver } from "../middleware/api-key";
import { NotFoundError } from "../lib/errors";
import { eq, and, sql } from "drizzle-orm";
import { nanoid } from "nanoid";
import { DocumentModels, CommonModels } from "./models";
import { getProjectDb } from "../db/project-db-helpers";
import {
  projectCollections,
  projectDocuments,
} from "../db/project-schema";
import { logProjectDatabaseOperation } from "../lib/project-logger-utils";
import { bunstoreEvents } from "../lib/bunstore-events";

// Helper function to get collection by name
async function getCollectionByName(
  projectDb: Awaited<ReturnType<typeof getProjectDb>>,
  collectionName: string,
) {
  const [collection] = await projectDb
    .select()
    .from(projectCollections)
    .where(eq(projectCollections.name, collectionName))
    .limit(1);

  return collection;
}

// Helper function to get collection by path
async function getCollectionByPath(
  projectDb: Awaited<ReturnType<typeof getProjectDb>>,
  collectionPath: string,
) {
  const [collection] = await projectDb
    .select()
    .from(projectCollections)
    .where(eq(projectCollections.path, collectionPath))
    .limit(1);

  return collection;
}

// Helper function to generate document path
function generateDocumentPath(
  collectionPath: string,
  documentId: string,
): string {
  return `${collectionPath}/${documentId}`;
}


export const dbRoutes = new Elysia({ prefix: "/db" })
  .resolve(apiKeyResolver)
  .model({
    "document.create": DocumentModels.create,
    "document.update": DocumentModels.update,
    "document.patch": DocumentModels.patch,
    "document.query": DocumentModels.query,
    "document.response": DocumentModels.response,
    "document.listResponse": DocumentModels.listResponse,
    "document.batch": DocumentModels.batch,
    "document.batchResponse": DocumentModels.batchResponse,
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
    // Handle generic errors
    if (error instanceof Error) {
      set.status = 500;
      return {
        error: {
          message: error.message,
          code: "INTERNAL_ERROR",
        },
      };
    }
  })
  // List all collections
  .get(
    "/collections",
    async ({ apiKey }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collections = await projectDb
        .select()
        .from(projectCollections);

      return collections.map((col) => ({
        collectionId: col.collectionId,
        name: col.name,
        path: col.path,
        parentDocumentId: col.parentDocumentId,
        parentPath: col.parentPath,
        createdAt: col.createdAt,
        updatedAt: col.updatedAt,
      }));
    },
    {
      response: {
        200: t.Array(
          t.Object({
            collectionId: t.String(),
            name: t.String(),
            path: t.String(),
            parentDocumentId: t.Nullable(t.String()),
            parentPath: t.Nullable(t.String()),
            createdAt: t.Date(),
            updatedAt: t.Date(),
          }),
        ),
      },
    },
  )
  // Create collection
  .post(
    "/collections",
    async ({ apiKey, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collectionId = nanoid();
      const path = body.name; // Simple path for root collections

      // Check if collection already exists
      const [existing] = await projectDb
        .select()
        .from(projectCollections)
        .where(eq(projectCollections.path, path))
        .limit(1);

      if (existing) {
        throw new Error("Collection with this name already exists");
      }

      const [collection] = await projectDb
        .insert(projectCollections)
        .values({
          collectionId: collectionId,
          name: body.name,
          path: path,
          parentDocumentId: null,
          parentPath: null,
        })
        .returning();

      if (!collection) {
        throw new Error("Failed to create collection");
      }

      logProjectDatabaseOperation(apiKey.projectId, "create", "collection", {
        collectionId: collection.collectionId,
        collectionName: collection.name,
      });

      return {
        collectionId: collection.collectionId,
        name: collection.name,
        path: collection.path,
        parentDocumentId: collection.parentDocumentId,
        parentPath: collection.parentPath,
        createdAt: collection.createdAt,
        updatedAt: collection.updatedAt,
      };
    },
    {
      body: t.Object({
        name: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          collectionId: t.String(),
          name: t.String(),
          path: t.String(),
          parentDocumentId: t.Nullable(t.String()),
          parentPath: t.Nullable(t.String()),
          createdAt: t.Date(),
          updatedAt: t.Date(),
        }),
      },
    },
  )
  // Get collection info by name
  .get(
    "/collections/:name",
    async ({ apiKey, params }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.name,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.name);
      }

      return {
        collectionId: collection.collectionId,
        name: collection.name,
        path: collection.path,
        parentDocumentId: collection.parentDocumentId,
        parentPath: collection.parentPath,
        createdAt: collection.createdAt,
        updatedAt: collection.updatedAt,
      };
    },
    {
      params: t.Object({
        name: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          collectionId: t.String(),
          name: t.String(),
          path: t.String(),
          parentDocumentId: t.Nullable(t.String()),
          parentPath: t.Nullable(t.String()),
          createdAt: t.Date(),
          updatedAt: t.Date(),
        }),
      },
    },
  )
  // Delete collection
  .delete(
    "/collections/:name",
    async ({ apiKey, params }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.name,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.name);
      }

      await projectDb
        .delete(projectCollections)
        .where(eq(projectCollections.collectionId, collection.collectionId));

      logProjectDatabaseOperation(apiKey.projectId, "delete", "collection", {
        collectionId: collection.collectionId,
        collectionName: collection.name,
      });

      return {
        message: "Collection deleted successfully",
      };
    },
    {
      params: t.Object({
        name: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Update collection schema
  .put(
    "/collections/:name/schema",
    async ({ apiKey, params, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.name,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.name);
      }

      // TODO: Store schema in collection metadata
      // For now, just return success
      return {
        message: "Schema updated successfully",
      };
    },
    {
      params: t.Object({
        name: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        schema: t.Record(t.String(), t.Any()),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Query documents in collection
  .get(
    "/:collection",
    async ({ apiKey, params, query }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      const limit = query.limit || 50;
      const offset = query.offset || 0;

      // Get all documents first (for filtering)
      let allDocs = await projectDb
        .select()
        .from(projectDocuments)
        .where(eq(projectDocuments.collectionId, collection.collectionId));

      // Apply filters in-memory
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
        collection: t.String({ minLength: 1 }),
      }),
      query: DocumentModels.query,
      response: {
        200: DocumentModels.listResponse,
      },
    },
  )
  // Create document
  .post(
    "/:collection",
    async ({ apiKey, params, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      // Try to get existing collection, or create it if it doesn't exist (Firebase-style)
      let collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        // Auto-create collection if it doesn't exist (Firebase behavior)
        const collectionId = nanoid();
        const path = params.collection; // Root collection path is just the name

        // Check if path already exists (shouldn't, but just in case)
        const [existing] = await projectDb
          .select()
          .from(projectCollections)
          .where(eq(projectCollections.path, path))
          .limit(1);

        if (existing) {
          collection = existing;
        } else {
          const [newCollection] = await projectDb
            .insert(projectCollections)
            .values({
              collectionId: collectionId,
              name: params.collection,
              path: path,
              parentDocumentId: null,
              parentPath: null,
            })
            .returning();

          if (!newCollection) {
            throw new Error("Failed to create collection");
          }

          collection = newCollection;
        }
      }

      const documentId = nanoid();
      const path = generateDocumentPath(collection.path, documentId);

      const [document] = await projectDb
        .insert(projectDocuments)
        .values({
          documentId: documentId,
          collectionId: collection.collectionId,
          path: path,
          data: body.data,
        })
        .returning();

      if (!document) {
        throw new Error("Failed to create document");
      }

      logProjectDatabaseOperation(apiKey.projectId, "create", "document", {
        documentId: document.documentId,
        collectionId: collection.collectionId,
        collectionName: params.collection,
      });

      // Emit document created event
      bunstoreEvents.emitCreated({
        projectId: apiKey.projectId,
        collectionPath: collection.path,
        documentId: document.documentId,
        path: document.path,
        data: document.data,
        createdAt: document.createdAt,
        updatedAt: document.updatedAt,
      });

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
      params: t.Object({
        collection: t.String({ minLength: 1 }),
      }),
      body: DocumentModels.create,
      response: {
        200: t.Object({
          data: DocumentModels.response,
        }),
      },
    },
  )
  // Get document by ID
  .get(
    "/:collection/:id",
    async ({ apiKey, params }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      const [document] = await projectDb
        .select()
        .from(projectDocuments)
        .where(
          and(
            eq(projectDocuments.documentId, params.id),
            eq(projectDocuments.collectionId, collection.collectionId),
          ),
        )
        .limit(1);

      if (!document) {
        throw new NotFoundError("Document", params.id);
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
      params: t.Object({
        collection: t.String({ minLength: 1 }),
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: DocumentModels.response,
        }),
      },
    },
  )
  // Upsert document (create if doesn't exist, update if exists)
  .put(
    "/:collection/:id",
    async ({ apiKey, params, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      // Try to get existing collection, or create it if it doesn't exist (Firebase-style)
      let collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        // Auto-create collection if it doesn't exist (Firebase behavior)
        const collectionId = nanoid();
        const path = params.collection; // Root collection path is just the name

        // Check if path already exists (shouldn't, but just in case)
        const [existing] = await projectDb
          .select()
          .from(projectCollections)
          .where(eq(projectCollections.path, path))
          .limit(1);

        if (existing) {
          collection = existing;
        } else {
          const [newCollection] = await projectDb
            .insert(projectCollections)
            .values({
              collectionId: collectionId,
              name: params.collection,
              path: path,
              parentDocumentId: null,
              parentPath: null,
            })
            .returning();

          if (!newCollection) {
            throw new Error("Failed to create collection");
          }

          collection = newCollection;
        }
      }

      // Check if document exists
      const [existingDocument] = await projectDb
        .select()
        .from(projectDocuments)
        .where(
          and(
            eq(projectDocuments.documentId, params.id),
            eq(projectDocuments.collectionId, collection.collectionId),
          ),
        )
        .limit(1);

      let result;
      const now = new Date();

      if (existingDocument) {
        // Update existing document
        const [updated] = await projectDb
          .update(projectDocuments)
          .set({
            data: body.data,
            updatedAt: now,
          })
          .where(eq(projectDocuments.documentId, params.id))
          .returning();

        if (!updated) {
          throw new Error("Failed to update document");
        }

        logProjectDatabaseOperation(apiKey.projectId, "update", "document", {
          documentId: updated.documentId,
          collectionId: collection.collectionId,
          collectionName: params.collection,
        });

        // Emit document updated event with old and new data
        bunstoreEvents.emitUpdated({
          projectId: apiKey.projectId,
          collectionPath: collection.path,
          documentId: updated.documentId,
          path: updated.path,
          data: updated.data,
          oldData: existingDocument.data,
          createdAt: updated.createdAt,
          updatedAt: updated.updatedAt,
        });

        result = updated;
      } else {
        // Create new document
        const path = generateDocumentPath(collection.path, params.id);

        const [created] = await projectDb
          .insert(projectDocuments)
          .values({
            documentId: params.id,
            collectionId: collection.collectionId,
            path: path,
            data: body.data,
          })
          .returning();

        if (!created) {
          throw new Error("Failed to create document");
        }

        logProjectDatabaseOperation(apiKey.projectId, "create", "document", {
          documentId: created.documentId,
          collectionId: collection.collectionId,
          collectionName: params.collection,
        });

        // Emit document created event
        bunstoreEvents.emitCreated({
          projectId: apiKey.projectId,
          collectionPath: collection.path,
          documentId: created.documentId,
          path: created.path,
          data: created.data,
          createdAt: created.createdAt,
          updatedAt: created.updatedAt,
        });

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
      params: t.Object({
        collection: t.String({ minLength: 1 }),
        id: t.String({ minLength: 1 }),
      }),
      body: DocumentModels.update,
      response: {
        200: t.Object({
          data: DocumentModels.response,
        }),
      },
    },
  )
  // Partial update document
  .patch(
    "/:collection/:id",
    async ({ apiKey, params, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      const [document] = await projectDb
        .select()
        .from(projectDocuments)
        .where(
          and(
            eq(projectDocuments.documentId, params.id),
            eq(projectDocuments.collectionId, collection.collectionId),
          ),
        )
        .limit(1);

      if (!document) {
        throw new NotFoundError("Document", params.id);
      }

      // Merge existing data with new data
      const mergedData = { ...document.data, ...body.data };

      const [updated] = await projectDb
        .update(projectDocuments)
        .set({
          data: mergedData,
          updatedAt: new Date(),
        })
        .where(eq(projectDocuments.documentId, params.id))
        .returning();

      if (!updated) {
        throw new Error("Failed to update document");
      }

      logProjectDatabaseOperation(apiKey.projectId, "patch", "document", {
        documentId: updated.documentId,
        collectionId: collection.collectionId,
        collectionName: params.collection,
      });

      // Emit document updated event with old and new data
      bunstoreEvents.emitUpdated({
        projectId: apiKey.projectId,
        collectionPath: collection.path,
        documentId: updated.documentId,
        path: updated.path,
        data: updated.data,
        oldData: document.data,
        createdAt: updated.createdAt,
        updatedAt: updated.updatedAt,
      });

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
      params: t.Object({
        collection: t.String({ minLength: 1 }),
        id: t.String({ minLength: 1 }),
      }),
      body: DocumentModels.patch,
      response: {
        200: t.Object({
          data: DocumentModels.response,
        }),
      },
    },
  )
  // Delete document
  .delete(
    "/:collection/:id",
    async ({ apiKey, params }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      const [document] = await projectDb
        .select()
        .from(projectDocuments)
        .where(
          and(
            eq(projectDocuments.documentId, params.id),
            eq(projectDocuments.collectionId, collection.collectionId),
          ),
        )
        .limit(1);

      if (!document) {
        throw new NotFoundError("Document", params.id);
      }

      // Emit document deleted event before deletion
      bunstoreEvents.emitDeleted({
        projectId: apiKey.projectId,
        collectionPath: collection.path,
        documentId: document.documentId,
        path: document.path,
        data: document.data,
        createdAt: document.createdAt,
        updatedAt: document.updatedAt,
      });

      await projectDb
        .delete(projectDocuments)
        .where(eq(projectDocuments.documentId, params.id));

      logProjectDatabaseOperation(apiKey.projectId, "delete", "document", {
        documentId: params.id,
        collectionId: collection.collectionId,
        collectionName: params.collection,
      });

      return {
        message: "Document deleted successfully",
      };
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Batch operations
  .post(
    "/:collection/batch",
    async ({ apiKey, params, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      const results: Array<{
        success: boolean;
        documentId?: string;
        error?: string;
        data?: Record<string, any>;
      }> = [];

      // Use a transaction for batch operations to improve performance
      // SQLite with WAL mode handles concurrent transactions well
      await projectDb.transaction(async (tx) => {
        // Process each operation in the batch
        for (const operation of body.operations) {
          try {
            if (operation.type === "create") {
              const documentId = nanoid();
              const path = generateDocumentPath(collection.path, documentId);

              const [newDocument] = await tx
                .insert(projectDocuments)
                .values({
                  documentId,
                  collectionId: collection.collectionId,
                  path,
                  data: operation.data || {},
                })
                .returning();

              if (newDocument) {
                // Emit document created event (after transaction commits)
                bunstoreEvents.emitCreated({
                  projectId: apiKey.projectId,
                  collectionPath: collection.path,
                  documentId: newDocument.documentId,
                  path: newDocument.path,
                  data: newDocument.data,
                  createdAt: newDocument.createdAt,
                  updatedAt: newDocument.updatedAt,
                });

                results.push({
                  success: true,
                  documentId: newDocument.documentId,
                  data: newDocument.data,
                });
              } else {
                results.push({
                  success: false,
                  error: "Failed to create document",
                });
              }
            } else if (operation.type === "update") {
              if (!operation.documentId) {
                results.push({
                  success: false,
                  error: "documentId is required for update operations",
                });
                continue;
              }

              // Get old document data before update
              const [oldDocument] = await tx
                .select()
                .from(projectDocuments)
                .where(eq(projectDocuments.documentId, operation.documentId))
                .limit(1);

              const [updated] = await tx
                .update(projectDocuments)
                .set({
                  data: operation.data || {},
                  updatedAt: new Date(),
                })
                .where(eq(projectDocuments.documentId, operation.documentId))
                .returning();

              if (updated) {
                // Emit document updated event
                bunstoreEvents.emitUpdated({
                  projectId: apiKey.projectId,
                  collectionPath: collection.path,
                  documentId: updated.documentId,
                  path: updated.path,
                  data: updated.data,
                  oldData: oldDocument?.data,
                  createdAt: updated.createdAt,
                  updatedAt: updated.updatedAt,
                });

                results.push({
                  success: true,
                  documentId: updated.documentId,
                  data: updated.data,
                });
              } else {
                results.push({
                  success: false,
                  documentId: operation.documentId,
                  error: "Document not found",
                });
              }
            } else if (operation.type === "delete") {
              if (!operation.documentId) {
                results.push({
                  success: false,
                  error: "documentId is required for delete operations",
                });
                continue;
              }

              // Get document data before deletion
              const [docToDelete] = await tx
                .select()
                .from(projectDocuments)
                .where(eq(projectDocuments.documentId, operation.documentId))
                .limit(1);

              if (docToDelete) {
                // Emit document deleted event before deletion
                bunstoreEvents.emitDeleted({
                  projectId: apiKey.projectId,
                  collectionPath: collection.path,
                  documentId: docToDelete.documentId,
                  path: docToDelete.path,
                  data: docToDelete.data,
                  createdAt: docToDelete.createdAt,
                  updatedAt: docToDelete.updatedAt,
                });
              }

              await tx
                .delete(projectDocuments)
                .where(eq(projectDocuments.documentId, operation.documentId));

              results.push({
                success: true,
                documentId: operation.documentId,
              });
            }
          } catch (error) {
            results.push({
              success: false,
              documentId: operation.documentId,
              error: error instanceof Error ? error.message : "Unknown error",
            });
          }
        }
      });

      // Log batch operation
      logProjectDatabaseOperation(apiKey.projectId, "batch", "document", {
        collectionId: collection.collectionId,
        collectionName: params.collection,
        operationCount: body.operations.length,
        successCount: results.filter((r) => r.success).length,
        errorCount: results.filter((r) => !r.success).length,
      });

      return {
        results,
        successCount: results.filter((r) => r.success).length,
        errorCount: results.filter((r) => !r.success).length,
      };
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
      }),
      body: DocumentModels.batch,
      response: {
        200: DocumentModels.batchResponse,
      },
    },
  )
  // Import data (placeholder)
  .post(
    "/:collection/import",
    async ({ apiKey, params, body }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      // TODO: Implement import logic
      return {
        message: "Import initiated",
        imported: 0,
      };
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        data: t.Array(t.Record(t.String(), t.Any())),
      }),
      response: {
        200: t.Object({
          message: t.String(),
          imported: t.Number(),
        }),
      },
    },
  )
  // Export data (placeholder)
  .get(
    "/:collection/export",
    async ({ apiKey, params }) => {
      const projectDb = await getProjectDb(apiKey.projectId);

      const collection = await getCollectionByName(
        projectDb,
        params.collection,
      );

      if (!collection) {
        throw new NotFoundError("Collection", params.collection);
      }

      // TODO: Implement export logic
      const documents = await projectDb
        .select()
        .from(projectDocuments)
        .where(eq(projectDocuments.collectionId, collection.collectionId));

      return {
        data: documents.map((doc) => doc.data),
      };
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: t.Array(t.Record(t.String(), t.Any())),
        }),
      },
    },
  )
  // List indexes (placeholder)
  .get(
    "/:collection/indexes",
    async ({ apiKey, params }) => {
      // TODO: Implement indexes
      return [];
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Array(t.Any()),
      },
    },
  )
  // Create index (placeholder)
  .post(
    "/:collection/indexes",
    async ({ apiKey, params, body }) => {

      // TODO: Implement index creation
      return {
        id: nanoid(),
        message: "Index created",
      };
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        fields: t.Array(t.String()),
      }),
      response: {
        200: t.Object({
          id: t.String(),
          message: t.String(),
        }),
      },
    },
  )
  // Delete index (placeholder)
  .delete(
    "/:collection/indexes/:id",
    async ({ apiKey, params }) => {
      // TODO: Implement index deletion
      return {
        message: "Index deleted",
      };
    },
    {
      params: t.Object({
        collection: t.String({ minLength: 1 }),
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Begin transaction (placeholder)
  .post(
    "/transactions/begin",
    async ({ apiKey }) => {
      // TODO: Implement transaction management
      const transactionId = nanoid();
      return {
        id: transactionId,
        message: "Transaction begun",
      };
    },
    {
      response: {
        200: t.Object({
          id: t.String(),
          message: t.String(),
        }),
      },
    },
  )
  // Commit transaction (placeholder)
  .post(
    "/transactions/:id/commit",
    async ({ apiKey, params }) => {
      // TODO: Implement transaction commit
      return {
        message: "Transaction committed",
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Rollback transaction (placeholder)
  .post(
    "/transactions/:id/rollback",
    async ({ apiKey, params }) => {
      // TODO: Implement transaction rollback
      return {
        message: "Transaction rolled back",
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  );
