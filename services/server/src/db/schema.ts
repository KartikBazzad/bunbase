import {
  pgTable,
  text,
  timestamp,
  boolean,
  jsonb,
  pgEnum,
} from "drizzle-orm/pg-core";
import { relations } from "drizzle-orm";

// Auth provider enum
export const authProviderEnum = pgEnum("auth_provider", [
  "email",
  "google",
  "github",
  "facebook",
  "apple",
]);

// Users table (aligned with better-auth structure)
export const users = pgTable("user", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: boolean("emailVerified").notNull().default(false),
  image: text("image"),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Sessions table (aligned with better-auth structure)
export const sessions = pgTable("session", {
  id: text("id").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  token: text("token").notNull().unique(),
  expiresAt: timestamp("expiresAt").notNull(),
  ipAddress: text("ipAddress"),
  userAgent: text("userAgent"),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// User accounts table (aligned with better-auth structure)
export const userAccounts = pgTable("userAccount", {
  id: text("id").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
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

// Verifications table (aligned with better-auth structure)
export const verifications = pgTable("verification", {
  id: text("id").primaryKey(),
  identifier: text("identifier").notNull(),
  value: text("value").notNull(),
  expiresAt: timestamp("expiresAt").notNull(),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Projects table
export const projects = pgTable("project", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  description: text("description").notNull(),
  ownerId: text("ownerId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Applications table
export const applications = pgTable("application", {
  id: text("id").primaryKey(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  name: text("name").notNull(),
  description: text("description").notNull(),
  type: text("type").notNull().default("web"),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Application API Keys table
export const applicationApiKeys = pgTable("applicationApiKey", {
  id: text("id").primaryKey(),
  applicationId: text("applicationId")
    .notNull()
    .references(() => applications.id, { onDelete: "cascade" }),
  keyHash: text("keyHash").notNull(), // Hashed API key for validation
  keyPrefix: text("keyPrefix").notNull(), // First 12 chars for display (e.g., "bunbase_pk_")
  keySuffix: text("keySuffix").notNull(), // Last 4 chars for identification
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  lastUsedAt: timestamp("lastUsedAt"),
  revokedAt: timestamp("revokedAt"),
});

// Databases table
export const databases = pgTable("database", {
  databaseId: text("databaseId").primaryKey(),
  name: text("name").notNull(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Project auth configuration table
export const projectAuth = pgTable("projectAuth", {
  projectId: text("projectId")
    .primaryKey()
    .references(() => projects.id, { onDelete: "cascade" }),
  providers: jsonb("providers").$type<string[]>().notNull().default(["email"]),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Storage table
export const storage = pgTable("storage", {
  storageId: text("storageId").primaryKey(),
  name: text("name").notNull(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Collections table (Firestore-like hierarchical structure)
export const collections = pgTable("collection", {
  collectionId: text("collectionId").primaryKey(),
  databaseId: text("databaseId")
    .notNull()
    .references(() => databases.databaseId, { onDelete: "cascade" }),
  name: text("name").notNull(),
  path: text("path").notNull().unique(), // Full path like "users" or "users/{userId}/posts"
  parentDocumentId: text("parentDocumentId"), // Nullable, for subcollections
  parentPath: text("parentPath"), // Nullable, path of parent collection
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Documents table (Firestore-like document storage)
export const documents = pgTable("document", {
  documentId: text("documentId").primaryKey(),
  collectionId: text("collectionId")
    .notNull()
    .references(() => collections.collectionId, { onDelete: "cascade" }),
  path: text("path").notNull().unique(), // Full document path like "users/{userId}" or "users/{userId}/posts/{postId}"
  data: jsonb("data").notNull().$type<Record<string, any>>(), // Document data as JSONB
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// Relations
export const usersRelations = relations(users, ({ many, one }) => ({
  sessions: many(sessions),
  accounts: many(userAccounts),
  projects: many(projects),
}));

export const sessionsRelations = relations(sessions, ({ one }) => ({
  user: one(users, {
    fields: [sessions.userId],
    references: [users.id],
  }),
}));

export const userAccountsRelations = relations(userAccounts, ({ one }) => ({
  user: one(users, {
    fields: [userAccounts.userId],
    references: [users.id],
  }),
}));

export const projectsRelations = relations(projects, ({ one, many }) => ({
  owner: one(users, {
    fields: [projects.ownerId],
    references: [users.id],
  }),
  applications: many(applications),
  databases: many(databases),
  storage: many(storage),
  auth: one(projectAuth, {
    fields: [projects.id],
    references: [projectAuth.projectId],
  }),
}));

export const applicationsRelations = relations(
  applications,
  ({ one, many }) => ({
    project: one(projects, {
      fields: [applications.projectId],
      references: [projects.id],
    }),
    apiKeys: many(applicationApiKeys),
  }),
);

export const applicationApiKeysRelations = relations(
  applicationApiKeys,
  ({ one }) => ({
    application: one(applications, {
      fields: [applicationApiKeys.applicationId],
      references: [applications.id],
    }),
  }),
);

export const databasesRelations = relations(databases, ({ one, many }) => ({
  project: one(projects, {
    fields: [databases.projectId],
    references: [projects.id],
  }),
  collections: many(collections),
}));

export const projectAuthRelations = relations(projectAuth, ({ one }) => ({
  project: one(projects, {
    fields: [projectAuth.projectId],
    references: [projects.id],
  }),
}));

export const storageRelations = relations(storage, ({ one }) => ({
  project: one(projects, {
    fields: [storage.projectId],
    references: [projects.id],
  }),
}));

export const collectionsRelations = relations(collections, ({ one, many }) => ({
  database: one(databases, {
    fields: [collections.databaseId],
    references: [databases.databaseId],
  }),
  parentDocument: one(documents, {
    fields: [collections.parentDocumentId],
    references: [documents.documentId],
  }),
  documents: many(documents),
}));

export const documentsRelations = relations(documents, ({ one, many }) => ({
  collection: one(collections, {
    fields: [documents.collectionId],
    references: [collections.collectionId],
  }),
  subcollections: many(collections),
}));
