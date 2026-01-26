import {
  pgTable,
  text,
  timestamp,
  boolean,
  jsonb,
  pgEnum,
  integer,
  unique,
} from "drizzle-orm/pg-core";
import { relations } from "drizzle-orm";
import { nanoid } from "nanoid";

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
  isBanned: boolean("isBanned").notNull().default(false),
  banReason: text("banReason"),
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
  isDefault: boolean("isDefault").notNull().default(false), // Default database for the project
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

// Project auth settings table
export const projectAuthSettings = pgTable("projectAuthSettings", {
  projectId: text("projectId")
    .primaryKey()
    .references(() => projects.id, { onDelete: "cascade" }),
  requireEmailVerification: boolean("requireEmailVerification")
    .notNull()
    .default(false),
  rateLimitMax: integer("rateLimitMax").notNull().default(5),
  rateLimitWindow: integer("rateLimitWindow").notNull().default(15),
  sessionExpirationDays: integer("sessionExpirationDays").notNull().default(30),
  minPasswordLength: integer("minPasswordLength").notNull().default(8),
  requireUppercase: boolean("requireUppercase").notNull().default(false),
  requireLowercase: boolean("requireLowercase").notNull().default(false),
  requireNumbers: boolean("requireNumbers").notNull().default(false),
  requireSpecialChars: boolean("requireSpecialChars").notNull().default(false),
  mfaEnabled: boolean("mfaEnabled").notNull().default(false),
  mfaRequired: boolean("mfaRequired").notNull().default(false),
  createdAt: timestamp("createdAt").notNull().defaultNow(),
  updatedAt: timestamp("updatedAt").notNull().defaultNow(),
});

// OAuth provider credentials table
export const projectOAuthProviders = pgTable("projectOAuthProvider", {
  id: text("id")
    .primaryKey()
    .$defaultFn(() => nanoid()),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  provider: authProviderEnum("provider").notNull(), // google, github, etc.
  clientId: text("clientId").notNull(),
  clientSecret: text("clientSecret").notNull(), // Encrypted
  redirectUri: text("redirectUri"),
  scopes: jsonb("scopes").$type<string[]>().default([]),
  isConfigured: boolean("isConfigured").notNull().default(false),
  lastTestedAt: timestamp("lastTestedAt"),
  lastTestStatus: text("lastTestStatus"), // "success" | "failed" | null
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
  functions: many(functions),
  auth: one(projectAuth, {
    fields: [projects.id],
    references: [projectAuth.projectId],
  }),
  authSettings: one(projectAuthSettings, {
    fields: [projects.id],
    references: [projectAuthSettings.projectId],
  }),
  oauthProviders: many(projectOAuthProviders),
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

export const projectAuthSettingsRelations = relations(
  projectAuthSettings,
  ({ one }) => ({
    project: one(projects, {
      fields: [projectAuthSettings.projectId],
      references: [projects.id],
    }),
  }),
);

export const projectOAuthProvidersRelations = relations(
  projectOAuthProviders,
  ({ one }) => ({
    project: one(projects, {
      fields: [projectOAuthProviders.projectId],
      references: [projects.id],
    }),
  }),
);

export const storageRelations = relations(storage, ({ one }) => ({
  project: one(projects, {
    fields: [storage.projectId],
    references: [projects.id],
  }),
}));

// Function status enum
export const functionStatusEnum = pgEnum("function_status", [
  "draft",
  "deployed",
  "paused",
]);

// Functions table
export const functions = pgTable(
  "function",
  {
    id: text("id").primaryKey(),
    projectId: text("projectId")
      .notNull()
      .references(() => projects.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    runtime: text("runtime").notNull().default("bun"),
    handler: text("handler").notNull(),
    status: functionStatusEnum("status").notNull().default("draft"),
    memory: integer("memory").default(512), // MB
    timeout: integer("timeout").default(30), // seconds
    maxConcurrentExecutions: integer("maxConcurrentExecutions").default(10),
    runtimeType: text("runtimeType").default("worker"), // worker | process
    activeVersionId: text("activeVersionId").references(
      () => functionVersions.id,
      { onDelete: "set null" },
    ),
    createdAt: timestamp("createdAt").notNull().defaultNow(),
    updatedAt: timestamp("updatedAt").notNull().defaultNow(),
  },
  (table) => ({
    projectNameUnique: unique().on(table.projectId, table.name),
  }),
);

// Function versions table
export const functionVersions = pgTable(
  "functionVersion",
  {
    id: text("id").primaryKey(),
    functionId: text("functionId")
      .notNull()
      .references(() => functions.id, { onDelete: "cascade" }),
    version: text("version").notNull(),
    codeHash: text("codeHash").notNull(),
    codePath: text("codePath").notNull(),
    deployedAt: timestamp("deployedAt"),
    createdAt: timestamp("createdAt").notNull().defaultNow(),
  },
  (table) => ({
    functionVersionUnique: unique().on(table.functionId, table.version),
  }),
);

// Function deployments table
export const functionDeployments = pgTable("functionDeployment", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  versionId: text("versionId")
    .notNull()
    .references(() => functionVersions.id, { onDelete: "cascade" }),
  environment: text("environment").notNull().default("production"),
  status: text("status").notNull().default("active"),
  deployedAt: timestamp("deployedAt").notNull().defaultNow(),
});

// Function environment variables table (encrypted)
export const functionEnvironments = pgTable(
  "functionEnvironment",
  {
    id: text("id").primaryKey(),
    functionId: text("functionId")
      .notNull()
      .references(() => functions.id, { onDelete: "cascade" }),
    key: text("key").notNull(),
    value: text("value").notNull(), // encrypted
    isSecret: boolean("isSecret").notNull().default(false),
    createdAt: timestamp("createdAt").notNull().defaultNow(),
  },
  (table) => ({
    functionKeyUnique: unique().on(table.functionId, table.key),
  }),
);

// Function logs table
export const functionLogs = pgTable("functionLog", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  executionId: text("executionId").notNull(),
  level: text("level").notNull(), // debug, info, warn, error
  message: text("message").notNull(),
  metadata: jsonb("metadata").$type<Record<string, any>>(),
  timestamp: timestamp("timestamp").notNull().defaultNow(),
});

// Function metrics table (aggregated daily)
export const functionMetrics = pgTable(
  "functionMetric",
  {
    id: text("id").primaryKey(),
    functionId: text("functionId")
      .notNull()
      .references(() => functions.id, { onDelete: "cascade" }),
    date: timestamp("date").notNull(),
    invocations: integer("invocations").notNull().default(0),
    errors: integer("errors").notNull().default(0),
    totalDuration: integer("totalDuration").notNull().default(0), // milliseconds
    coldStarts: integer("coldStarts").notNull().default(0),
  },
  (table) => ({
    functionDateUnique: unique().on(table.functionId, table.date),
  }),
);

// Function metrics table (minute-level, for real-time analysis)
export const functionMetricsMinute = pgTable(
  "functionMetricMinute",
  {
    id: text("id").primaryKey(),
    functionId: text("functionId")
      .notNull()
      .references(() => functions.id, { onDelete: "cascade" }),
    timestamp: timestamp("timestamp").notNull(), // Rounded to minute
    invocations: integer("invocations").notNull().default(0),
    errors: integer("errors").notNull().default(0),
    totalDuration: integer("totalDuration").notNull().default(0), // milliseconds
    coldStarts: integer("coldStarts").notNull().default(0),
  },
  (table) => ({
    functionTimestampUnique: unique().on(table.functionId, table.timestamp),
  }),
);

// Function relations
export const functionsRelations = relations(functions, ({ one, many }) => ({
  project: one(projects, {
    fields: [functions.projectId],
    references: [projects.id],
  }),
  versions: many(functionVersions),
  deployments: many(functionDeployments),
  environments: many(functionEnvironments),
  logs: many(functionLogs),
  metrics: many(functionMetrics),
}));

export const functionVersionsRelations = relations(
  functionVersions,
  ({ one, many }) => ({
    function: one(functions, {
      fields: [functionVersions.functionId],
      references: [functions.id],
    }),
    deployments: many(functionDeployments),
  }),
);

export const functionDeploymentsRelations = relations(
  functionDeployments,
  ({ one }) => ({
    function: one(functions, {
      fields: [functionDeployments.functionId],
      references: [functions.id],
    }),
    version: one(functionVersions, {
      fields: [functionDeployments.versionId],
      references: [functionVersions.id],
    }),
  }),
);

export const functionEnvironmentsRelations = relations(
  functionEnvironments,
  ({ one }) => ({
    function: one(functions, {
      fields: [functionEnvironments.functionId],
      references: [functions.id],
    }),
  }),
);

export const functionLogsRelations = relations(functionLogs, ({ one }) => ({
  function: one(functions, {
    fields: [functionLogs.functionId],
    references: [functions.id],
  }),
}));

export const functionMetricsRelations = relations(functionMetrics, ({ one }) => ({
  function: one(functions, {
    fields: [functionMetrics.functionId],
    references: [functions.id],
  }),
}));

export const functionMetricsMinuteRelations = relations(
  functionMetricsMinute,
  ({ one }) => ({
    function: one(functions, {
      fields: [functionMetricsMinute.functionId],
      references: [functions.id],
    }),
  }),
);
