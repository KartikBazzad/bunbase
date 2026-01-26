import { sqliteTable, text, integer } from "drizzle-orm/sqlite-core";
import { relations } from "drizzle-orm";

// Users table (Better Auth compatible - for project-specific users)
export const projectUsers = sqliteTable("user", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: integer("emailVerified", { mode: "boolean" })
    .notNull()
    .default(false),
  image: text("image"),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Accounts table (Better Auth compatible - for project-specific accounts)
export const projectAccounts = sqliteTable("account", {
  id: text("id").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => projectUsers.id, { onDelete: "cascade" }),
  accountId: text("accountId").notNull(), // The ID of the account as provided by the SSO or equal to userId for credential accounts
  providerId: text("providerId").notNull(),
  accessToken: text("accessToken"),
  refreshToken: text("refreshToken"),
  accessTokenExpiresAt: integer("accessTokenExpiresAt", { mode: "timestamp" }),
  refreshTokenExpiresAt: integer("refreshTokenExpiresAt", {
    mode: "timestamp",
  }),
  scope: text("scope"),
  idToken: text("idToken"),
  password: text("password"), // hashed password
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// AuthSettings table (project authentication configuration)
export const authSettings = sqliteTable("authSettings", {
  id: text("id").primaryKey(),
  providers: text("providers", { mode: "json" })
    .$type<string[]>()
    .notNull()
    .default(["email"]),
  emailAndPassword: text("emailAndPassword", { mode: "json" }).$type<{
    enabled: boolean;
    requireEmailVerification?: boolean;
  }>(),
  socialProviders: text("socialProviders", { mode: "json" }).$type<
    Record<string, any>
  >(),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// OAuth Providers table (for project-specific OAuth configuration)
export const oauthProviders = sqliteTable("oauthProvider", {
  id: text("id").primaryKey(),
  provider: text("provider").notNull(), // google, github, etc. - validated as one of: email, google, github, facebook, apple
  clientId: text("clientId").notNull(),
  clientSecret: text("clientSecret").notNull(), // Encrypted
  redirectUri: text("redirectUri"),
  scopes: text("scopes", { mode: "json" }).$type<string[]>().default([]),
  isConfigured: integer("isConfigured", { mode: "boolean" })
    .notNull()
    .default(false),
  lastTestedAt: integer("lastTestedAt", { mode: "timestamp" }),
  lastTestStatus: text("lastTestStatus"), // "success" | "failed" | null
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Detailed Auth Settings table (password requirements, MFA, etc.)
export const detailedAuthSettings = sqliteTable("detailedAuthSettings", {
  id: text("id").primaryKey().default("default"),
  requireEmailVerification: integer("requireEmailVerification", {
    mode: "boolean",
  })
    .notNull()
    .default(false),
  rateLimitMax: text("rateLimitMax").notNull().default("5"), // Store as text for large numbers
  rateLimitWindow: text("rateLimitWindow").notNull().default("15"), // Store as text for large numbers
  sessionExpirationDays: text("sessionExpirationDays").notNull().default("30"), // Store as text for large numbers
  minPasswordLength: text("minPasswordLength").notNull().default("8"), // Store as text for large numbers
  requireUppercase: integer("requireUppercase", { mode: "boolean" })
    .notNull()
    .default(false),
  requireLowercase: integer("requireLowercase", { mode: "boolean" })
    .notNull()
    .default(false),
  requireNumbers: integer("requireNumbers", { mode: "boolean" })
    .notNull()
    .default(false),
  requireSpecialChars: integer("requireSpecialChars", { mode: "boolean" })
    .notNull()
    .default(false),
  mfaEnabled: integer("mfaEnabled", { mode: "boolean" })
    .notNull()
    .default(false),
  mfaRequired: integer("mfaRequired", { mode: "boolean" })
    .notNull()
    .default(false),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Collections table (Firestore-like hierarchical structure)
// Note: databaseId column kept for backward compatibility but not used
export const projectCollections = sqliteTable("collection", {
  collectionId: text("collectionId").primaryKey(),
  databaseId: text("databaseId"), // Kept for backward compatibility, not used
  name: text("name").notNull(),
  path: text("path").notNull().unique(), // Full path like "users" or "users/{userId}/posts"
  parentDocumentId: text("parentDocumentId"), // Nullable, for subcollections
  parentPath: text("parentPath"), // Nullable, path of parent collection
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Documents table (Firestore-like document storage)
export const projectDocuments = sqliteTable("document", {
  documentId: text("documentId").primaryKey(),
  collectionId: text("collectionId")
    .notNull()
    .references(() => projectCollections.collectionId, { onDelete: "cascade" }),
  path: text("path").notNull().unique(), // Full document path like "users/{userId}" or "users/{userId}/posts/{postId}"
  data: text("data", { mode: "json" }).notNull().$type<Record<string, any>>(), // Document data as JSON
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Files table (storage file metadata)
export const projectFiles = sqliteTable("file", {
  fileId: text("fileId").primaryKey(),
  bucketId: text("bucketId").notNull(), // References storage.storageId
  path: text("path").notNull(), // File path within bucket
  size: text("size").notNull(), // File size in bytes (as text for large numbers)
  mimeType: text("mimeType").notNull(),
  metadata: text("metadata", { mode: "json" })
    .$type<Record<string, any>>()
    .default({}),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Relations
export const projectUsersRelations = relations(projectUsers, ({ many }) => ({
  accounts: many(projectAccounts),
}));

export const projectAccountsRelations = relations(
  projectAccounts,
  ({ one }) => ({
    user: one(projectUsers, {
      fields: [projectAccounts.userId],
      references: [projectUsers.id],
    }),
  }),
);

export const projectCollectionsRelations = relations(
  projectCollections,
  ({ one, many }) => ({
    parentDocument: one(projectDocuments, {
      fields: [projectCollections.parentDocumentId],
      references: [projectDocuments.documentId],
    }),
    documents: many(projectDocuments),
  }),
);

export const projectDocumentsRelations = relations(
  projectDocuments,
  ({ one, many }) => ({
    collection: one(projectCollections, {
      fields: [projectDocuments.collectionId],
      references: [projectCollections.collectionId],
    }),
    subcollections: many(projectCollections),
  }),
);
