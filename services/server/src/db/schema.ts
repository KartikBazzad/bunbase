import { sqliteTable, text, integer, unique } from "drizzle-orm/sqlite-core";
import { relations } from "drizzle-orm";
import { nanoid } from "nanoid";

// Users table (aligned with better-auth structure)
export const users = sqliteTable("user", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: integer("emailVerified", { mode: "boolean" })
    .notNull()
    .default(false),
  image: text("image"),
  isBanned: integer("isBanned", { mode: "boolean" }).notNull().default(false),
  banReason: text("banReason"),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Sessions table (aligned with better-auth structure)
export const sessions = sqliteTable("session", {
  id: text("id").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  token: text("token").notNull().unique(),
  expiresAt: integer("expiresAt", { mode: "timestamp" }).notNull(),
  ipAddress: text("ipAddress"),
  userAgent: text("userAgent"),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// User accounts table (aligned with better-auth structure)
export const userAccounts = sqliteTable("userAccount", {
  id: text("id").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
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

// Verifications table (aligned with better-auth structure)
export const verifications = sqliteTable("verification", {
  id: text("id").primaryKey(),
  identifier: text("identifier").notNull(),
  value: text("value").notNull(),
  expiresAt: integer("expiresAt", { mode: "timestamp" }).notNull(),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Projects table
export const projects = sqliteTable("project", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  description: text("description").notNull(),
  ownerId: text("ownerId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Applications table
export const applications = sqliteTable("application", {
  id: text("id").primaryKey(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  name: text("name").notNull(),
  description: text("description").notNull(),
  type: text("type").notNull().default("web"),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Application API Keys table
export const applicationApiKeys = sqliteTable("applicationApiKey", {
  id: text("id").primaryKey(),
  applicationId: text("applicationId")
    .notNull()
    .references(() => applications.id, { onDelete: "cascade" }),
  keyHash: text("keyHash").notNull(), // Hashed API key for validation
  keyPrefix: text("keyPrefix").notNull(), // First 12 chars for display (e.g., "bunbase_pk_")
  keySuffix: text("keySuffix").notNull(), // Last 4 chars for identification
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  lastUsedAt: integer("lastUsedAt", { mode: "timestamp" }),
  revokedAt: integer("revokedAt", { mode: "timestamp" }),
});

// Databases table
export const databases = sqliteTable("database", {
  databaseId: text("databaseId").primaryKey(),
  name: text("name").notNull(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  isDefault: integer("isDefault", { mode: "boolean" }).notNull().default(false), // Default database for the project
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Project auth configuration table
export const projectAuth = sqliteTable("projectAuth", {
  projectId: text("projectId")
    .primaryKey()
    .references(() => projects.id, { onDelete: "cascade" }),
  providers: text("providers", { mode: "json" })
    .$type<string[]>()
    .notNull()
    .default(["email"]),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Project auth settings table
export const projectAuthSettings = sqliteTable("projectAuthSettings", {
  projectId: text("projectId")
    .primaryKey()
    .references(() => projects.id, { onDelete: "cascade" }),
  requireEmailVerification: integer("requireEmailVerification", {
    mode: "boolean",
  })
    .notNull()
    .default(false),
  rateLimitMax: integer("rateLimitMax").notNull().default(5),
  rateLimitWindow: integer("rateLimitWindow").notNull().default(15),
  sessionExpirationDays: integer("sessionExpirationDays").notNull().default(30),
  minPasswordLength: integer("minPasswordLength").notNull().default(8),
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

// OAuth provider credentials table
export const projectOAuthProviders = sqliteTable("projectOAuthProvider", {
  id: text("id")
    .primaryKey()
    .$defaultFn(() => nanoid()),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
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

// Storage table
export const storage = sqliteTable("storage", {
  storageId: text("storageId").primaryKey(),
  name: text("name").notNull(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Collections table (Firestore-like hierarchical structure)
export const collections = sqliteTable("collection", {
  collectionId: text("collectionId").primaryKey(),
  databaseId: text("databaseId")
    .notNull()
    .references(() => databases.databaseId, { onDelete: "cascade" }),
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

// Functions table
export const functions = sqliteTable("function", {
  id: text("id").primaryKey(),
  projectId: text("projectId")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  name: text("name").notNull(),
  runtime: text("runtime").notNull().default("bun"),
  handler: text("handler").notNull(),
  status: text("status").notNull().default("draft"), // draft, deployed, paused
  memory: integer("memory").default(512), // MB
  timeout: integer("timeout").default(30), // seconds
  maxConcurrentExecutions: integer("maxConcurrentExecutions").default(10),
  runtimeType: text("runtimeType").default("worker"), // worker | process
  activeVersionId: text("activeVersionId").references(
    () => functionVersions.id,
    {
      onDelete: "set null",
    },
  ),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: integer("updatedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Function versions table
export const functionVersions = sqliteTable("functionVersion", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  version: text("version").notNull(),
  codeHash: text("codeHash").notNull(),
  codePath: text("codePath").notNull(),
  deployedAt: integer("deployedAt", { mode: "timestamp" }),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Function deployments table
export const functionDeployments = sqliteTable("functionDeployment", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  versionId: text("versionId")
    .notNull()
    .references(() => functionVersions.id, { onDelete: "cascade" }),
  environment: text("environment").notNull().default("production"),
  status: text("status").notNull().default("active"),
  deployedAt: integer("deployedAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Function environment variables table (encrypted)
export const functionEnvironments = sqliteTable("functionEnvironment", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  key: text("key").notNull(),
  value: text("value").notNull(), // encrypted
  isSecret: integer("isSecret", { mode: "boolean" }).notNull().default(false),
  createdAt: integer("createdAt", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Function logs table
export const functionLogs = sqliteTable("functionLog", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  executionId: text("executionId").notNull(),
  level: text("level").notNull(), // debug, info, warn, error
  message: text("message").notNull(),
  metadata: text("metadata", { mode: "json" }).$type<Record<string, any>>(),
  timestamp: integer("timestamp", { mode: "timestamp" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Function metrics table (aggregated daily)
export const functionMetrics = sqliteTable("functionMetric", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  date: integer("date", { mode: "timestamp" }).notNull(),
  invocations: integer("invocations").notNull().default(0),
  errors: integer("errors").notNull().default(0),
  totalDuration: integer("totalDuration").notNull().default(0), // milliseconds
  coldStarts: integer("coldStarts").notNull().default(0),
});

// Function metrics table (minute-level, for real-time analysis)
export const functionMetricsMinute = sqliteTable("functionMetricMinute", {
  id: text("id").primaryKey(),
  functionId: text("functionId")
    .notNull()
    .references(() => functions.id, { onDelete: "cascade" }),
  timestamp: integer("timestamp", { mode: "timestamp" }).notNull(), // Rounded to minute
  invocations: integer("invocations").notNull().default(0),
  errors: integer("errors").notNull().default(0),
  totalDuration: integer("totalDuration").notNull().default(0), // milliseconds
  coldStarts: integer("coldStarts").notNull().default(0),
});

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

export const functionMetricsRelations = relations(
  functionMetrics,
  ({ one }) => ({
    function: one(functions, {
      fields: [functionMetrics.functionId],
      references: [functions.id],
    }),
  }),
);

export const functionMetricsMinuteRelations = relations(
  functionMetricsMinute,
  ({ one }) => ({
    function: one(functions, {
      fields: [functionMetricsMinute.functionId],
      references: [functions.id],
    }),
  }),
);
