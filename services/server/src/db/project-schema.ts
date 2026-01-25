import {
  pgTable,
  text,
  timestamp,
  boolean,
  jsonb,
  pgEnum,
} from "drizzle-orm/pg-core";
import { relations } from "drizzle-orm";

// Auth provider enum (same as backend)
export const authProviderEnum = pgEnum("auth_provider", [
  "email",
  "google",
  "github",
  "facebook",
  "apple",
]);

// Users table (Better Auth compatible - for project-specific users)
export const projectUsers = pgTable("user", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: boolean("emailVerified").notNull().default(false),
  image: text("image"),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Accounts table (Better Auth compatible - for project-specific accounts)
export const projectAccounts = pgTable("account", {
  id: text("id").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => projectUsers.id, { onDelete: "cascade" }),
  accountId: text("accountId").notNull(), // The ID of the account as provided by the SSO or equal to userId for credential accounts
  providerId: text("providerId").notNull(),
  accessToken: text("accessToken"),
  refreshToken: text("refreshToken"),
  accessTokenExpiresAt: timestamp("accessTokenExpiresAt"),
  refreshTokenExpiresAt: timestamp("refreshTokenExpiresAt"),
  scope: text("scope"),
  idToken: text("idToken"),
  password: text("password"), // hashed password
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// AuthSettings table (project authentication configuration)
export const authSettings = pgTable("authSettings", {
  id: text("id").primaryKey(),
  providers: jsonb("providers").$type<string[]>().notNull().default(["email"]),
  emailAndPassword: jsonb("emailAndPassword").$type<{
    enabled: boolean;
    requireEmailVerification?: boolean;
  }>(),
  socialProviders: jsonb("socialProviders").$type<Record<string, any>>(),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Databases table (user-created databases within the project)
export const projectDatabases = pgTable("database", {
  databaseId: text("databaseId").primaryKey(),
  name: text("name").notNull(),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Collections table (Firestore-like hierarchical structure)
export const projectCollections = pgTable("collection", {
  collectionId: text("collectionId").primaryKey(),
  databaseId: text("databaseId")
    .notNull()
    .references(() => projectDatabases.databaseId, { onDelete: "cascade" }),
  name: text("name").notNull(),
  path: text("path").notNull().unique(), // Full path like "users" or "users/{userId}/posts"
  parentDocumentId: text("parentDocumentId"), // Nullable, for subcollections
  parentPath: text("parentPath"), // Nullable, path of parent collection
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Documents table (Firestore-like document storage)
export const projectDocuments = pgTable("document", {
  documentId: text("documentId").primaryKey(),
  collectionId: text("collectionId")
    .notNull()
    .references(() => projectCollections.collectionId, { onDelete: "cascade" }),
  path: text("path").notNull().unique(), // Full document path like "users/{userId}" or "users/{userId}/posts/{postId}"
  data: jsonb("data").notNull().$type<Record<string, any>>(), // Document data as JSONB
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Files table (storage file metadata)
export const projectFiles = pgTable("file", {
  fileId: text("fileId").primaryKey(),
  bucketId: text("bucketId").notNull(), // References storage.storageId
  path: text("path").notNull(), // File path within bucket
  size: text("size").notNull(), // File size in bytes (as text for large numbers)
  mimeType: text("mimeType").notNull(),
  metadata: jsonb("metadata").$type<Record<string, any>>().default({}),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
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

export const projectDatabasesRelations = relations(
  projectDatabases,
  ({ many }) => ({
    collections: many(projectCollections),
  }),
);

export const projectCollectionsRelations = relations(
  projectCollections,
  ({ one, many }) => ({
    database: one(projectDatabases, {
      fields: [projectCollections.databaseId],
      references: [projectDatabases.databaseId],
    }),
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
