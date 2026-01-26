import { Elysia, t } from "elysia";
import { db, projects, storage } from "../db";
import { authResolver } from "../middleware/auth";
import { apiKeyResolver } from "../middleware/api-key";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq, and } from "drizzle-orm";
import { nanoid } from "nanoid";
import { StorageBucketModels, StorageFileModels } from "./storage-models";
import { CommonModels } from "./models";
import { join } from "path";
import { logProjectStorageOperation } from "../lib/project-logger-utils";

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
  const dirFile = Bun.file(bucketDir);
  if (!(await dirFile.exists())) {
    // Create directory by writing a keep file
    await Bun.write(join(bucketDir, ".keep"), "");
  }
  return bucketDir;
}

// Note: storageRoutes removed - all storage routes are now in storageApiRoutes
// which supports both API key and session-based authentication
export const storageRoutes = new Elysia({ prefix: "/storage" });

// Storage Routes (matching requirements)
// These routes support both API key and session-based authentication
// API keys are used by client SDKs, sessions are used by dashboard
export const storageApiRoutes = new Elysia({ prefix: "/storage" })
  .resolve(apiKeyResolver)
  .model({
    "storage.bucket.create": StorageBucketModels.create,
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
    async ({ apiKey }) => {
      const buckets = await db
        .select()
        .from(storage)
        .where(eq(storage.projectId, apiKey.projectId));

      return buckets.map((bucket) => ({
        storageId: bucket.storageId,
        name: bucket.name,
        projectId: bucket.projectId,
        createdAt: bucket.createdAt,
        updatedAt: bucket.updatedAt,
      }));
    },
    {
      response: {
        200: StorageBucketModels.listResponse,
      },
    },
  )
  // Create bucket
  .post(
    "/buckets",
    async ({ apiKey, body }) => {
      const bucketId = nanoid();
      const [newBucket] = await db
        .insert(storage)
        .values({
          storageId: bucketId,
          name: body.name,
          projectId: apiKey.projectId,
          createdAt: new Date(),
          updatedAt: new Date(),
        })
        .returning();

      // Create storage directory for bucket
      await ensureStorageDir(bucketId);

      logProjectStorageOperation(apiKey.projectId, "create_bucket", bucketId, {
        bucketId: newBucket.storageId,
        bucketName: newBucket.name,
      });

      return {
        storageId: newBucket.storageId,
        name: newBucket.name,
        projectId: newBucket.projectId,
        createdAt: newBucket.createdAt,
        updatedAt: newBucket.updatedAt,
      };
    },
    {
      body: StorageBucketModels.create,
      response: {
        200: StorageBucketModels.response,
      },
    },
  )
  // Get bucket info by name
  .get(
    "/buckets/:name",
    async ({ apiKey, params }) => {
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.name),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.name);
      }

      return {
        storageId: bucket.storageId,
        name: bucket.name,
        projectId: bucket.projectId,
        createdAt: bucket.createdAt,
        updatedAt: bucket.updatedAt,
      };
    },
    {
      params: t.Object({
        name: t.String({ minLength: 1 }),
      }),
      response: {
        200: StorageBucketModels.response,
      },
    },
  )
  // Update bucket config
  .put(
    "/buckets/:name/config",
    async ({ apiKey, params, body }) => {
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.name),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.name);
      }

      // TODO: Add config field to storage table if needed
      // For now, just update the bucket
      const [updated] = await db
        .update(storage)
        .set({
          updatedAt: new Date(),
        })
        .where(eq(storage.storageId, bucket.storageId))
        .returning();

      return {
        storageId: updated.storageId,
        name: updated.name,
        projectId: updated.projectId,
        createdAt: updated.createdAt,
        updatedAt: updated.updatedAt,
      };
    },
    {
      params: t.Object({
        name: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        // Config fields can be added here
      }),
      response: {
        200: StorageBucketModels.response,
      },
    },
  )
  // Delete bucket by name
  .delete(
    "/buckets/:name",
    async ({ apiKey, params }) => {
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.name),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.name);
      }

      await db.delete(storage).where(eq(storage.storageId, bucket.storageId));

      logProjectStorageOperation(
        apiKey.projectId,
        "delete_bucket",
        bucket.storageId,
        {
          bucketId: bucket.storageId,
          bucketName: bucket.name,
        },
      );

      return {
        message: "Bucket deleted successfully",
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
  // List files in bucket
  .get(
    "/:bucket",
    async ({ apiKey, params, query }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      // TODO: Implement file listing from project database
      // For now, return empty list
      return {
        files: [],
        total: 0,
        limit: query.limit || 50,
        offset: query.offset || 0,
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
      }),
      query: StorageFileModels.list,
      response: {
        200: StorageFileModels.listResponse,
      },
    },
  )
  // Upload file
  .post(
    "/:bucket/upload",
    async ({ apiKey, params, body }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

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
      const parentDirFile = Bun.file(parentDir);
      if (!(await parentDirFile.exists())) {
        // Create directory by writing a keep file
        await Bun.write(join(parentDir, ".keep"), "");
      }

      // Write file to disk using Bun.write
      const arrayBuffer = await file.arrayBuffer();
      await Bun.write(fullPath, arrayBuffer);

      // Get file size using Bun.file
      const fileHandle = Bun.file(fullPath);
      // Read the file to get size (Bun.file.size is only valid after reading)
      await fileHandle.arrayBuffer();
      const fileSize = fileHandle.size;

      logProjectStorageOperation(apiKey.projectId, "upload", fileId, {
        fileId,
        bucketId: bucket.storageId,
        bucketName: params.bucket,
        path: sanitizedPath,
        size: fileSize,
        mimeType: file.type || "application/octet-stream",
      });

      return {
        fileId,
        bucketId: bucket.storageId,
        path: sanitizedPath,
        size: fileSize,
        mimeType: file.type || "application/octet-stream",
        metadata: body.metadata || {},
        createdAt: new Date(),
        updatedAt: new Date(),
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        file: t.File({
          maxSize: "5gb",
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
    "/:bucket/:path",
    async ({ apiKey, params, set }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const fullPath = join(bucketDir, params.path);
      const fileHandle = Bun.file(fullPath);

      if (!(await fileHandle.exists())) {
        throw new NotFoundError("File", params.path);
      }

      // Read file using Bun.file
      const fileBuffer = await fileHandle.arrayBuffer();
      // Get file size (size is available after reading)
      const fileSize = fileHandle.size;

      logProjectStorageOperation(apiKey.projectId, "download", undefined, {
        bucketId: bucket.storageId,
        bucketName: params.bucket,
        path: params.path,
        size: fileSize,
      });

      // Set appropriate headers
      set.headers["Content-Type"] = "application/octet-stream";
      set.headers["Content-Length"] = fileSize.toString();
      set.headers["Content-Disposition"] =
        `attachment; filename="${params.path.split("/").pop()}"`;

      return fileBuffer;
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
    },
  )
  // Update file metadata
  .put(
    "/:bucket/:path",
    async ({ apiKey, params, body }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const fullPath = join(bucketDir, params.path);

      if (!existsSync(fullPath)) {
        throw new NotFoundError("File", params.path);
      }

      // TODO: Store metadata in project database
      // For now, just return success
      return {
        message: "File metadata updated successfully",
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        metadata: t.Optional(t.Record(t.String(), t.Any())),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Delete file
  .delete(
    "/:bucket/:path",
    async ({ apiKey, params }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const fullPath = join(bucketDir, params.path);
      const fileHandle = Bun.file(fullPath);

      if (!(await fileHandle.exists())) {
        throw new NotFoundError("File", params.path);
      }

      // Delete file - Bun doesn't have direct unlink, use fs/promises as fallback
      const { unlink } = await import("fs/promises");
      await unlink(fullPath);

      logProjectStorageOperation(apiKey.projectId, "delete", undefined, {
        bucketId: bucket.storageId,
        bucketName: params.bucket,
        path: params.path,
      });

      return {
        message: "File deleted successfully",
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Copy file
  .post(
    "/:bucket/:path/copy",
    async ({ apiKey, params, body }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const sourcePath = join(bucketDir, params.path);
      const destPath = join(bucketDir, body.destination);
      const sourceFile = Bun.file(sourcePath);

      if (!(await sourceFile.exists())) {
        throw new NotFoundError("File", params.path);
      }

      // Copy file using Bun.write
      await Bun.write(destPath, sourceFile);

      logProjectStorageOperation(apiKey.projectId, "copy", undefined, {
        bucketId: bucket.storageId,
        bucketName: params.bucket,
        sourcePath: params.path,
        destinationPath: body.destination,
      });

      return {
        message: "File copied successfully",
        destination: body.destination,
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        destination: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          message: t.String(),
          destination: t.String(),
        }),
      },
    },
  )
  // Move file
  .post(
    "/:bucket/:path/move",
    async ({ apiKey, params, body }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      const bucketDir = await ensureStorageDir(bucket.storageId);
      const sourcePath = join(bucketDir, params.path);
      const destPath = join(bucketDir, body.destination);
      const sourceFile = Bun.file(sourcePath);

      if (!(await sourceFile.exists())) {
        throw new NotFoundError("File", params.path);
      }

      // Move file (copy then delete) using Bun APIs
      await Bun.write(destPath, sourceFile);
      const { unlink } = await import("fs/promises");
      await unlink(sourcePath);

      logProjectStorageOperation(apiKey.projectId, "move", undefined, {
        bucketId: bucket.storageId,
        bucketName: params.bucket,
        sourcePath: params.path,
        destinationPath: body.destination,
      });

      return {
        message: "File moved successfully",
        destination: body.destination,
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        destination: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          message: t.String(),
          destination: t.String(),
        }),
      },
    },
  )
  // Generate signed upload URL
  .post(
    "/:bucket/signed-url",
    async ({ apiKey, params, body }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      // TODO: Implement signed URL generation
      // For now, return a placeholder
      const expiresIn = body.expiresIn || 3600; // Default 1 hour
      const expiresAt = new Date(Date.now() + expiresIn * 1000);

      return {
        url: `/api/storage/${params.bucket}/upload?token=placeholder`,
        expiresAt,
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
      }),
      body: StorageFileModels.signedUrl,
      response: {
        200: StorageFileModels.signedUrlResponse,
      },
    },
  )
  // Generate signed download URL
  .post(
    "/:bucket/:path/signed",
    async ({ apiKey, params, body }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      // TODO: Implement signed URL generation
      // For now, return a placeholder
      const expiresIn = body.expiresIn || 3600; // Default 1 hour
      const expiresAt = new Date(Date.now() + expiresIn * 1000);

      return {
        url: `/api/storage/${params.bucket}/${params.path}?token=placeholder`,
        expiresAt,
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
        path: t.String({ minLength: 1 }),
      }),
      body: StorageFileModels.signedUrl,
      response: {
        200: StorageFileModels.signedUrlResponse,
      },
    },
  )
  // Multipart upload (placeholder - full implementation needed)
  .post(
    "/:bucket/multipart",
    async ({ apiKey, params }) => {
      // Find bucket by name
      const [bucket] = await db
        .select()
        .from(storage)
        .where(
          and(
            eq(storage.name, params.bucket),
            eq(storage.projectId, apiKey.projectId),
          ),
        )
        .limit(1);

      if (!bucket) {
        throw new NotFoundError("Bucket", params.bucket);
      }

      // TODO: Implement multipart upload
      return {
        uploadId: nanoid(),
        message: "Multipart upload initiated",
      };
    },
    {
      params: t.Object({
        bucket: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          uploadId: t.String(),
          message: t.String(),
        }),
      },
    },
  );
