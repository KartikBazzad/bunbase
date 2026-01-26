import { Database } from "bun:sqlite";
import { drizzle, type BunSQLiteDatabase } from "drizzle-orm/bun-sqlite";
import { join } from "path";
import { mkdir } from "fs/promises";
import { existsSync, mkdirSync } from "fs";
import * as projectSchema from "./project-schema";
import { sql } from "drizzle-orm";

type ProjectDbInstance = {
  sqlite: Database;
  db: BunSQLiteDatabase<typeof projectSchema>;
};

/**
 * Singleton manager for project-specific Bun.SQLite database instances.
 * Each project gets its own isolated database at databases/<projectId>/database.db
 */
export class ProjectDatabaseManager {
  private static instance: ProjectDatabaseManager;
  private instances: Map<string, ProjectDbInstance> = new Map();
  private basePath: string;

  private constructor() {
    // Resolve base path relative to server root (same level as src/)
    // import.meta.dir points to the directory containing this file
    // We need to go up to server root, then into databases/
    this.basePath = join(import.meta.dir, "../../databases");
    // Ensure base directory exists synchronously
    if (!existsSync(this.basePath)) {
      try {
        mkdirSync(this.basePath, { recursive: true });
      } catch (error) {
        console.error("Failed to create base databases directory:", error);
      }
    }
  }

  /**
   * Get the singleton instance
   */
  static getInstance(): ProjectDatabaseManager {
    if (!ProjectDatabaseManager.instance) {
      ProjectDatabaseManager.instance = new ProjectDatabaseManager();
    }
    return ProjectDatabaseManager.instance;
  }

  /**
   * Get or create a database instance for a project
   */
  async getInstance(
    projectId: string,
  ): Promise<BunSQLiteDatabase<typeof projectSchema>> {
    // Check if instance already exists
    const existing = this.instances.get(projectId);
    if (existing) {
      return existing.db;
    }

    // Create new instance
    return this.createInstance(projectId);
  }

  /**
   * Create a new database instance for a project
   */
  async createInstance(
    projectId: string,
  ): Promise<BunSQLiteDatabase<typeof projectSchema>> {
    // Check if already exists
    const existing = this.instances.get(projectId);
    if (existing) {
      return existing.db;
    }

    // Ensure database directory exists
    const projectDir = join(this.basePath, projectId);
    const dbPath = join(projectDir, "database.db");
    try {
      await this.ensureDirectory(projectDir);
    } catch (error) {
      console.error(
        `Failed to create directory for project ${projectId}:`,
        error,
      );
      throw new Error(
        `Failed to create database directory: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    }

    // Create Bun.SQLite database instance
    let sqlite: Database;
    try {
      sqlite = new Database(dbPath);
      // Optimize SQLite for high concurrency and performance
      sqlite.run(`
        PRAGMA journal_mode = WAL;
        PRAGMA synchronous = NORMAL;
        PRAGMA busy_timeout = 10000;
        PRAGMA foreign_keys = ON;
        PRAGMA cache_size = -64000; -- 64MB cache
        PRAGMA temp_store = MEMORY;
        PRAGMA mmap_size = 268435456; -- 256MB memory-mapped I/O
        PRAGMA page_size = 4096;
        PRAGMA optimize;
      `);
    } catch (error) {
      console.error(
        `Failed to create SQLite instance for project ${projectId}:`,
        error,
      );
      const errorMessage =
        error instanceof Error
          ? error.message
          : typeof error === "string"
            ? error
            : "Unknown error";
      throw new Error(`Failed to create SQLite instance: ${errorMessage}`);
    }

    // Create Drizzle instance
    const db = drizzle(sqlite, { schema: projectSchema });

    // Store instance
    this.instances.set(projectId, { sqlite, db });

    return db;
  }

  /**
   * Initialize a project database with schema
   */
  async initializeDatabase(projectId: string): Promise<void> {
    const db = await this.getInstance(projectId);

    // Check if tables already exist by querying sqlite_master
    // If the query fails or returns no results, we proceed with initialization
    let tablesExist = false;
    try {
      const tables = db.all(sql`
        SELECT name 
        FROM sqlite_master 
        WHERE type = 'table' 
        AND name IN ('user', 'account', 'authSettings', 'oauthProvider', 'detailedAuthSettings', 'collection', 'document')
      `);

      // If tables already exist, skip initialization
      if (tables.length > 0) {
        tablesExist = true;
      }
    } catch (error) {
      console.error("Failed to check if tables exist", error);
      // If query fails, assume tables don't exist and proceed with initialization
      tablesExist = false;
    }

    if (tablesExist) {
      return;
    }

    // Create users table
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "user" (
        "id" text PRIMARY KEY NOT NULL,
        "name" text NOT NULL,
        "email" text NOT NULL UNIQUE,
        "emailVerified" integer DEFAULT 0 NOT NULL,
        "image" text,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL
      )
    `);

    // Create accounts table
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "account" (
        "id" text PRIMARY KEY NOT NULL,
        "userId" text NOT NULL,
        "accountId" text NOT NULL,
        "providerId" text NOT NULL,
        "accessToken" text,
        "refreshToken" text,
        "accessTokenExpiresAt" integer,
        "refreshTokenExpiresAt" integer,
        "scope" text,
        "idToken" text,
        "password" text,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL,
        CONSTRAINT "account_userId_user_id_fk" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE cascade
      )
    `);

    // Create authSettings table
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "authSettings" (
        "id" text PRIMARY KEY NOT NULL,
        "providers" text DEFAULT '["email"]' NOT NULL,
        "emailAndPassword" text,
        "socialProviders" text,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL
      )
    `);

    // Create oauthProvider table
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "oauthProvider" (
        "id" text PRIMARY KEY NOT NULL,
        "provider" text NOT NULL,
        "clientId" text NOT NULL,
        "clientSecret" text NOT NULL,
        "redirectUri" text,
        "scopes" text DEFAULT '[]',
        "isConfigured" integer DEFAULT 0 NOT NULL,
        "lastTestedAt" integer,
        "lastTestStatus" text,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL
      )
    `);

    // Create detailedAuthSettings table
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "detailedAuthSettings" (
        "id" text PRIMARY KEY DEFAULT 'default' NOT NULL,
        "requireEmailVerification" integer DEFAULT 0 NOT NULL,
        "rateLimitMax" text DEFAULT '5' NOT NULL,
        "rateLimitWindow" text DEFAULT '15' NOT NULL,
        "sessionExpirationDays" text DEFAULT '30' NOT NULL,
        "minPasswordLength" text DEFAULT '8' NOT NULL,
        "requireUppercase" integer DEFAULT 0 NOT NULL,
        "requireLowercase" integer DEFAULT 0 NOT NULL,
        "requireNumbers" integer DEFAULT 0 NOT NULL,
        "requireSpecialChars" integer DEFAULT 0 NOT NULL,
        "mfaEnabled" integer DEFAULT 0 NOT NULL,
        "mfaRequired" integer DEFAULT 0 NOT NULL,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL
      )
    `);

    // Create collections table
    // Note: databaseId column kept for backward compatibility but foreign key removed
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "collection" (
        "collectionId" text PRIMARY KEY NOT NULL,
        "databaseId" text,
        "name" text NOT NULL,
        "path" text NOT NULL UNIQUE,
        "parentDocumentId" text,
        "parentPath" text,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL
      )
    `);

    // Create documents table
    db.run(sql`
      CREATE TABLE IF NOT EXISTS "document" (
        "documentId" text PRIMARY KEY NOT NULL,
        "collectionId" text NOT NULL,
        "path" text NOT NULL UNIQUE,
        "data" text NOT NULL,
        "createdAt" integer NOT NULL,
        "updatedAt" integer NOT NULL,
        CONSTRAINT "document_collectionId_collection_collectionId_fk" FOREIGN KEY ("collectionId") REFERENCES "collection"("collectionId") ON DELETE cascade
      )
    `);

    // Create indexes for better performance
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "account_userId_idx" ON "account"("userId")
    `);
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "document_collectionId_idx" ON "document"("collectionId")
    `);
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "collection_parentDocumentId_idx" ON "collection"("parentDocumentId")
    `);
    
    // Composite indexes for common query patterns
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "document_collectionId_documentId_idx" ON "document"("collectionId", "documentId")
    `);
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "collection_name_path_idx" ON "collection"("name", "path")
    `);
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "document_path_idx" ON "document"("path")
    `);
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "document_updatedAt_idx" ON "document"("updatedAt")
    `);
    db.run(sql`
      CREATE INDEX IF NOT EXISTS "collection_path_idx" ON "collection"("path")
    `);

    // Insert default auth settings
    db.run(sql`
      INSERT INTO "authSettings" ("id", "providers", "emailAndPassword", "createdAt", "updatedAt")
      VALUES ('default', '["email"]', '{"enabled": true, "requireEmailVerification": false}', ${Date.now()}, ${Date.now()})
      ON CONFLICT ("id") DO NOTHING
    `);

    // Insert default detailed auth settings
    db.run(sql`
      INSERT INTO "detailedAuthSettings" ("id", "createdAt", "updatedAt")
      VALUES ('default', ${Date.now()}, ${Date.now()})
      ON CONFLICT ("id") DO NOTHING
    `);
  }

  /**
   * Close a database instance
   */
  async closeInstance(projectId: string): Promise<void> {
    const instance = this.instances.get(projectId);
    if (instance) {
      instance.sqlite.close();
      this.instances.delete(projectId);
    }
  }

  /**
   * Close all database instances
   */
  async closeAll(): Promise<void> {
    const closePromises = Array.from(this.instances.keys()).map((projectId) =>
      this.closeInstance(projectId),
    );
    await Promise.all(closePromises);
  }

  /**
   * Check if a project database exists
   */
  hasInstance(projectId: string): boolean {
    return this.instances.has(projectId);
  }

  /**
   * Get the database path for a project
   */
  getDatabasePath(projectId: string): string {
    return join(this.basePath, projectId, "database.db");
  }

  /**
   * Ensure directory exists
   */
  private async ensureDirectory(path: string): Promise<void> {
    if (!existsSync(path)) {
      // Create directory recursively (creates parent directories if needed)
      await mkdir(path, { recursive: true });
    }
  }
}
