import { t } from "elysia";

// Storage Bucket Models
export const StorageBucketModels = {
  create: t.Object({
    name: t.String({
      minLength: 1,
      maxLength: 255,
      error: "Bucket name must be between 1 and 255 characters",
    }),
  }),
  params: t.Object({
    bucketId: t.String({
      minLength: 1,
      error: "Bucket ID is required",
    }),
  }),
  response: t.Object({
    storageId: t.String(),
    name: t.String(),
    projectId: t.String(),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Array(
    t.Object({
      storageId: t.String(),
      name: t.String(),
      projectId: t.String(),
      createdAt: t.Date(),
      updatedAt: t.Date(),
    }),
  ),
};

// File Models
export const StorageFileModels = {
  upload: t.Object({
    path: t.Optional(t.String()), // Optional file path, auto-generated if not provided
    metadata: t.Optional(t.Record(t.String(), t.Any())), // Optional metadata
  }),
  list: t.Object({
    prefix: t.Optional(t.String()), // List files with prefix
    limit: t.Optional(t.Number({ minimum: 1, maximum: 1000 })),
    offset: t.Optional(t.Number({ minimum: 0 })),
  }),
  fileResponse: t.Object({
    fileId: t.String(),
    bucketId: t.String(),
    path: t.String(),
    size: t.Number(),
    mimeType: t.String(),
    metadata: t.Record(t.String(), t.Any()),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Object({
    files: t.Array(
      t.Object({
        fileId: t.String(),
        bucketId: t.String(),
        path: t.String(),
        size: t.Number(),
        mimeType: t.String(),
        metadata: t.Record(t.String(), t.Any()),
        createdAt: t.Date(),
        updatedAt: t.Date(),
      }),
    ),
    total: t.Number(),
    limit: t.Number(),
    offset: t.Number(),
  }),
  signedUrl: t.Object({
    expiresIn: t.Optional(t.Number({ minimum: 1, maximum: 604800 })), // 1 second to 7 days, default 1 hour
  }),
  signedUrlResponse: t.Object({
    url: t.String(),
    expiresAt: t.Date(),
  }),
};
