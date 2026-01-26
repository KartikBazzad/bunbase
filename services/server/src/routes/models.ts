import { t } from "elysia";

// Project Models
export const ProjectModels = {
  create: t.Object({
    name: t.String({
      minLength: 1,
      maxLength: 255,
      error: "Name must be between 1 and 255 characters",
    }),
    description: t.String({
      minLength: 1,
      maxLength: 1000,
      error: "Description must be between 1 and 1000 characters",
    }),
  }),
  update: t.Object({
    name: t.Optional(
      t.String({
        minLength: 1,
        maxLength: 255,
        error: "Name must be between 1 and 255 characters",
      }),
    ),
    description: t.Optional(
      t.String({
        minLength: 1,
        maxLength: 1000,
        error: "Description must be between 1 and 1000 characters",
      }),
    ),
  }),
  params: t.Object({
    id: t.String({
      minLength: 1,
      error: "Project ID is required",
    }),
  }),
  response: t.Object({
    id: t.String(),
    name: t.String(),
    description: t.String(),
    ownerId: t.String(),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Array(
    t.Object({
      id: t.String(),
      name: t.String(),
      description: t.String(),
      ownerId: t.String(),
      createdAt: t.Date(),
      updatedAt: t.Date(),
    }),
  ),
};

// Application Models
export const ApplicationModels = {
  create: t.Object({
    name: t.String({
      minLength: 1,
      maxLength: 255,
      error: "Name must be between 1 and 255 characters",
    }),
    description: t.String({
      minLength: 1,
      maxLength: 1000,
      error: "Description must be between 1 and 1000 characters",
    }),
    type: t.Optional(t.Union([t.Literal("web")])),
  }),
  update: t.Object({
    name: t.Optional(
      t.String({
        minLength: 1,
        maxLength: 255,
        error: "Name must be between 1 and 255 characters",
      }),
    ),
    description: t.Optional(
      t.String({
        minLength: 1,
        maxLength: 1000,
        error: "Description must be between 1 and 1000 characters",
      }),
    ),
    type: t.Optional(t.Union([t.Literal("web")])),
  }),
  params: t.Object({
    id: t.String({
      minLength: 1,
      error: "Application ID is required",
    }),
    projectId: t.String({
      minLength: 1,
      error: "Project ID is required",
    }),
  }),
  response: t.Object({
    id: t.String(),
    projectId: t.String(),
    name: t.String(),
    description: t.String(),
    type: t.String(),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Array(
    t.Object({
      id: t.String(),
      projectId: t.String(),
      name: t.String(),
      description: t.String(),
      type: t.String(),
      createdAt: t.Date(),
      updatedAt: t.Date(),
    }),
  ),
};

// Database Models
export const DatabaseModels = {
  create: t.Object({
    name: t.String({
      minLength: 1,
      maxLength: 255,
      error: "Database name must be between 1 and 255 characters",
    }),
  }),
  params: t.Object({
    id: t.String({
      minLength: 1,
      error: "Database ID is required",
    }),
    projectId: t.String({
      minLength: 1,
      error: "Project ID is required",
    }),
  }),
  response: t.Object({
    databaseId: t.String(),
    name: t.String(),
    projectId: t.String(),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Array(
    t.Object({
      databaseId: t.String(),
      name: t.String(),
      projectId: t.String(),
      createdAt: t.Date(),
      updatedAt: t.Date(),
    }),
  ),
};

// Auth Provider Models
export const AuthProviderModels = {
  update: t.Object({
    providers: t.Array(
      t.Union([
        t.Literal("email"),
        t.Literal("google"),
        t.Literal("github"),
        t.Literal("facebook"),
        t.Literal("apple"),
      ]),
      {
        minItems: 1,
        error: "At least one provider must be specified",
      },
    ),
  }),
  params: t.Object({
    projectId: t.String({
      minLength: 1,
      error: "Project ID is required",
    }),
    provider: t.Union([
      t.Literal("email"),
      t.Literal("google"),
      t.Literal("github"),
      t.Literal("facebook"),
      t.Literal("apple"),
    ]),
  }),
  response: t.Object({
    projectId: t.String(),
    providers: t.Array(t.String()),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
};

// Collection Models
export const CollectionModels = {
  create: t.Object({
    name: t.String({
      minLength: 1,
      maxLength: 255,
      error: "Collection name must be between 1 and 255 characters",
    }),
    parentPath: t.Optional(t.String()), // For subcollections
    parentDocumentId: t.Optional(t.String()), // For subcollections
  }),
  update: t.Object({
    name: t.Optional(
      t.String({
        minLength: 1,
        maxLength: 255,
        error: "Collection name must be between 1 and 255 characters",
      }),
    ),
  }),
  params: t.Object({
    collectionId: t.String({
      minLength: 1,
      error: "Collection ID is required",
    }),
  }),
  response: t.Object({
    collectionId: t.String(),
    name: t.String(),
    path: t.String(),
    parentDocumentId: t.Nullable(t.String()),
    parentPath: t.Nullable(t.String()),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Array(
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
};

// Document Models
export const DocumentModels = {
  create: t.Object({
    data: t.Record(t.String(), t.Any()), // JSONB data
  }),
  update: t.Object({
    data: t.Record(t.String(), t.Any()), // Full document replacement
  }),
  patch: t.Object({
    data: t.Record(t.String(), t.Any()), // Partial update
  }),
  query: t.Object({
    collectionPath: t.Optional(t.String()),
    filter: t.Optional(t.Record(t.String(), t.Any())),
    sort: t.Optional(
      t.Record(t.String(), t.Union([t.Literal("asc"), t.Literal("desc")])),
    ),
    limit: t.Optional(t.Number({ minimum: 1, maximum: 1000 })),
    offset: t.Optional(t.Number({ minimum: 0 })),
  }),
  params: t.Object({
    databaseId: t.String({
      minLength: 1,
      error: "Database ID is required",
    }),
    collectionId: t.String({
      minLength: 1,
      error: "Collection ID is required",
    }),
    documentId: t.String({
      minLength: 1,
      error: "Document ID is required",
    }),
  }),
  response: t.Object({
    documentId: t.String(),
    collectionId: t.String(),
    path: t.String(),
    data: t.Record(t.String(), t.Any()),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Object({
    data: t.Array(
      t.Object({
        documentId: t.String(),
        collectionId: t.String(),
        path: t.String(),
        data: t.Record(t.String(), t.Any()),
        createdAt: t.Date(),
        updatedAt: t.Date(),
      }),
    ),
    total: t.Number(),
    limit: t.Number(),
    offset: t.Number(),
  }),
  batch: t.Object({
    operations: t.Array(
      t.Object({
        type: t.Union([
          t.Literal("create"),
          t.Literal("update"),
          t.Literal("upsert"),
          t.Literal("delete"),
        ]),
        documentId: t.Optional(t.String()), // Required for update/upsert/delete
        data: t.Optional(t.Record(t.String(), t.Any())), // Required for create/update/upsert
      }),
      {
        minItems: 1,
        maxItems: 1000, // Limit batch size
      },
    ),
  }),
  batchResponse: t.Object({
    results: t.Array(
      t.Object({
        success: t.Boolean(),
        documentId: t.Optional(t.String()),
        error: t.Optional(t.String()),
        data: t.Optional(t.Record(t.String(), t.Any())),
      }),
    ),
    successCount: t.Number(),
    errorCount: t.Number(),
  }),
  atomic: t.Object({
    operations: t.Array(
      t.Union([
        t.Object({
          type: t.Literal("increment"),
          field: t.String(),
          value: t.Number(),
        }),
        t.Object({
          type: t.Literal("decrement"),
          field: t.String(),
          value: t.Number(),
        }),
        t.Object({
          type: t.Literal("arrayPush"),
          field: t.String(),
          value: t.Any(),
        }),
        t.Object({
          type: t.Literal("arrayRemove"),
          field: t.String(),
          value: t.Any(),
        }),
        t.Object({
          type: t.Literal("set"),
          field: t.String(),
          value: t.Any(),
        }),
      ]),
    ),
  }),
  atomicResponse: t.Object({
    data: t.Record(t.String(), t.Any()),
    operations: t.Array(
      t.Object({
        type: t.String(),
        field: t.String(),
        success: t.Boolean(),
      }),
    ),
  }),
};

// Application API Key Models
export const ApplicationKeyModels = {
  params: t.Object({
    id: t.String({
      minLength: 1,
      error: "Application ID is required",
    }),
  }),
  response: t.Object({
    id: t.String(),
    applicationId: t.String(),
    keyPrefix: t.String(),
    keySuffix: t.String(),
    createdAt: t.Date(),
    lastUsedAt: t.Nullable(t.Date()),
    revokedAt: t.Nullable(t.Date()),
  }),
  generateResponse: t.Object({
    id: t.String(),
    applicationId: t.String(),
    key: t.String(), // Full key shown only once
    keyPrefix: t.String(),
    keySuffix: t.String(),
    createdAt: t.Date(),
  }),
};

// Function Models
export const FunctionModels = {
  create: t.Object({
    name: t.String({ minLength: 1, maxLength: 255 }),
    runtime: t.String({ default: "bun" }),
    handler: t.String({ minLength: 1 }),
    code: t.Optional(t.String()),
    memory: t.Optional(t.Number({ minimum: 128, maximum: 4096 })),
    timeout: t.Optional(t.Number({ minimum: 1, maximum: 900 })),
  }),
  update: t.Object({
    name: t.Optional(t.String({ minLength: 1, maxLength: 255 })),
    runtime: t.Optional(t.String()),
    handler: t.Optional(t.String({ minLength: 1 })),
    code: t.Optional(t.String()),
    memory: t.Optional(t.Number({ minimum: 128, maximum: 4096 })),
    timeout: t.Optional(t.Number({ minimum: 1, maximum: 900 })),
  }),
  params: t.Object({
    id: t.String({ minLength: 1 }),
  }),
  response: t.Object({
    id: t.String(),
    name: t.String(),
    runtime: t.String(),
    handler: t.String(),
    status: t.String(),
    memory: t.Optional(t.Number()),
    timeout: t.Optional(t.Number()),
    createdAt: t.Date(),
    updatedAt: t.Date(),
  }),
  listResponse: t.Array(
    t.Object({
      id: t.String(),
      name: t.String(),
      runtime: t.String(),
      handler: t.String(),
      status: t.String(),
      memory: t.Optional(t.Number()),
      timeout: t.Optional(t.Number()),
      createdAt: t.Date(),
      updatedAt: t.Date(),
    }),
  ),
  invoke: t.Object({
    body: t.Optional(t.Any()),
    headers: t.Optional(t.Record(t.String(), t.String())),
  }),
  env: t.Object({
    key: t.String({ minLength: 1 }),
    value: t.String(),
  }),
  deployResponse: t.Object({
    message: t.String(),
    version: t.String(),
    deploymentId: t.String(),
  }),
  metricsResponse: t.Object({
    invocations: t.Number(),
    errors: t.Number(),
    averageDuration: t.Number(),
    lastInvoked: t.Nullable(t.Date()),
  }),
};

// Common Response Models
export const CommonModels = {
  success: t.Object({
    message: t.String(),
  }),
  error: t.Object({
    error: t.Object({
      message: t.String(),
      code: t.Optional(t.String()),
      details: t.Optional(t.Any()),
    }),
  }),
};
