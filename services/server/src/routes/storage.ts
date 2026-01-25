import { Elysia, t } from "elysia";
import { db, projects, storage } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq } from "drizzle-orm";
import { nanoid } from "nanoid";
import { StorageBucketModels, StorageFileModels } from "./storage-models";
import { CommonModels } from "./models";
import { writeFile, readFile, mkdir, stat, unlink } from "fs/promises";
import { join } from "path";
import { existsSync } from "fs";

// Storage directory (can be configured via env)
const STORAGE_DIR = process.env.STORAGE_DIR || "./storage";

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

// Helper function to verify bucket access
async function verifyBucketAccess(
  bucketId: string,
  projectId: string,
  userId: string,
): Promise<{
  bucket: typeof storage.$inferSelect;
  project: typeof projects.$inferSelect;
}> {
  const project = await verifyProjectOwnership(projectId, userId);

  const [bucket] = await db
    .select()
    .from(storage)
    .where(eq(storage.storageId, bucketId))
    .limit(1);

  if (!bucket) {
    throw new NotFoundError("Bucket", bucketId);
  }

  if (bucket.projectId !== projectId) {
    throw new ForbiddenError("Bucket does not belong to this project");
  }

  return { bucket, project };
}

// Ensure storage directory exists
async function ensureStorageDir(bucketId: string): Promise<string> {
  const bucketDir = join(STORAGE_DIR, bucketId);
  if (!existsSync(bucketDir)) {
    await mkdir(bucketDir, { recursive: true });
  }
  return bucketDir;
}

export const storageRoutes = new Elysia({ prefix: "/storage" })
  .resolve(authResolver)
  .model({
    "storage.bucket.create": StorageBucketModels.create,
    "storage.bucket.params": StorageBucketModels.params,
    "storage.bucket.response": StorageBucketModels.response,
    "storage.bucket.listResponse": StorageBucketModels.listResponse,
    "storage.file.upload": StorageFileModels.upload,
    "storage.file.list": StorageFileModels.list,
    "storage.file.response": StorageFileModels.fileResponse,
    "storage.file.listResponse": StorageFileModels.listResponse,
    "storage.file.signedUrl": StorageFileModels.signedUrl,
    "storage.file.signedUrlResponse": StorageFileModels.signedUrlResponse,
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
  // List buckets
  .get(
    "/buckets",
    async ({ user, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyProjectOwnership(query.projectId as string, user.id);

      const buckets = await db
        .select()
        .from(storage)
        .where(eq(storage.projectId, query.projectId as string));

      return buckets.map((bucket) => ({
        storageId: bucket.storageId,
        name: bucket.name,
        projectId: bucket.projectId,
        createdAt: bucket.createdAt,
        updatedAt: bucket.updatedAt,
      }));
    },
    {
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: StorageBucketModels.listResponse,
      },
    },
  )
  // Create bucket
  .post(
    "/buckets",
    async ({ user, body, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyProjectOwnership(query.projectId as string, user.id);

      const bucketId = nanoid();
      const [newBucket] = await db
        .insert(storage)
        .values({
          storageId: bucketId,
          name: body.name,
          projectId: query.projectId as string,
          createdAt: new Date(),
          updatedAt: new Date(),
        })
        .returning();

      // Create storage directory for bucket
      await ensureStorageDir(bucketId);

      return {
        storageId: newBucket.storageId,
        name: newBucket.name,
        projectId: newBucket.projectId,
        createdAt: newBucket.createdAt,
        updatedAt: newBucket.updatedAt,
      };
    },
    {
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      body: StorageBucketModels.create,
      response: {
        200: StorageBucketModels.response,
      },
    },
  )
  // Get bucket info
  .get(
    "/buckets/:bucketId",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      const { bucket } = await verifyBucketAccess(
        params.bucketId,
        query.projectId as string,
        user.id,
      );

      return {
        storageId: bucket.storageId,
        name: bucket.name,
        projectId: bucket.projectId,
        createdAt: bucket.createdAt,
        updatedAt: bucket.updatedAt,
      };
    },
    {
      params: StorageBucketModels.params,
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: StorageBucketModels.response,
      },
    },
  )
  // Delete bucket
  .delete(
    "/buckets/:bucketId",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      await verifyBucketAccess(
        params.bucketId,
        query.projectId as string,
        user.id,
      );

      await db.delete(storage).where(eq(storage.storageId, params.bucketId));

      return {
        message: "Bucket deleted successfully",
      };
    },
    {
      params: StorageBucketModels.params,
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Upload file
  .post(
    "/buckets/:bucketId/upload",
    async ({ user, params, body, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      const { bucket } = await verifyBucketAccess(
        params.bucketId,
        query.projectId as string,
        user.id,
      );

      const file = body.file;
      if (!file) {
        throw new Error("File is required");
      }

      // Generate file path
      const fileId = nanoid();
      const filePath = body.path || file.name || `file-${fileId}`;
      const sanitizedPath = filePath.replace(/[^a-zA-Z0-9._/-]/g, "_");

      // Ensure storage directory exists
      const bucketDir = await ensureStorageDir(bucket.storageId);
      const fullPath = join(bucketDir, sanitizedPath);

      // Ensure parent directory exists
      const parentDir = join(fullPath, "..");
      if (!existsSync(parentDir)) {
        await mkdir(parentDir, { recursive: true });
      }

      // Write file to disk
      const arrayBuffer = await file.arrayBuffer();
      const buffer = Buffer.from(arrayBuffer);
      await writeFile(fullPath, buffer);

      // Get file stats
      const stats = await stat(fullPath);

      // TODO: Store file metadata in project database
      // For now, return file info
      return {
        fileId,
        bucketId: bucket.storageId,
        path: sanitizedPath,
        size: stats.size,
        mimeType: file.type || "application/octet-stream",
        metadata: body.metadata || {},
        createdAt: new Date(),
        updatedAt: new Date(),
      };
    },
    {
      params: StorageBucketModels.params,
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        file: t.File({
          maxSize: "5gb", // 5GB max file size
        }),
        path: t.Optional(t.String()),
        metadata: t.Optional(t.Record(t.String(), t.Any())),
      }),
      response: {
        200: StorageFileModels.fileResponse,
      },
    },
  )
  // Download file
  .get(
    "/buckets/:bucketId/files/*",
    async ({ user, params, query, set }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      const { bucket } = await verifyBucketAccess(
        params.bucketId,
        query.projectId as string,
        user.id,
      );

      // Get file path from wildcard
      const filePath = params["*"];
      if (!filePath) {
        throw new NotFoundError("File", "path not provided");
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const fullPath = join(bucketDir, filePath);

      if (!existsSync(fullPath)) {
        throw new NotFoundError("File", filePath);
      }

      // Read file
      const fileBuffer = await readFile(fullPath);
      const stats = await stat(fullPath);

      // Set appropriate headers
      set.headers["Content-Type"] = "application/octet-stream";
      set.headers["Content-Length"] = stats.size.toString();
      set.headers["Content-Disposition"] = `attachment; filename="${filePath.split("/").pop()}"`;

      return fileBuffer;
    },
    {
      params: t.Object({
        bucketId: t.String({ minLength: 1 }),
        "*": t.String(),
      }),
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
    },
  )
  // Delete file
  .delete(
    "/buckets/:bucketId/files/*",
    async ({ user, params, query }) => {
      requireAuth(user);
      if (!query.projectId) {
        throw new Error("projectId query parameter is required");
      }
      const { bucket } = await verifyBucketAccess(
        params.bucketId,
        query.projectId as string,
        user.id,
      );

      const filePath = params["*"];
      if (!filePath) {
        throw new NotFoundError("File", "path not provided");
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const fullPath = join(bucketDir, filePath);

      if (!existsSync(fullPath)) {
        throw new NotFoundError("File", filePath);
      }

      await unlink(fullPath);

      return {
        message: "File deleted successfully",
      };
    },
    {
      params: t.Object({
        bucketId: t.String({ minLength: 1 }),
        "*": t.String(),
      }),
      query: t.Object({
        projectId: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  );
