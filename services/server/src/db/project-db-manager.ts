import { PGlite } from "@electric-sql/pglite";
import { drizzle, type PgliteDatabase } from "drizzle-orm/pglite";
import { join } from "path";
import { mkdir } from "fs/promises";
import { existsSync } from "fs";
import * as projectSchema from "./project-schema";
import { sql } from "drizzle-orm";

type ProjectDbInstance = {
  pglite: PGlite;
  db: PgliteDatabase<typeof projectSchema>;
};

/**
 * Singleton manager for project-specific PGLite database instances.
 * Each project gets its own isolated database at databases/<projectId>/pgdata
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
  ): Promise<PgliteDatabase<typeof projectSchema>> {
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
  ): Promise<PgliteDatabase<typeof projectSchema>> {
    // Check if already exists
    const existing = this.instances.get(projectId);
    if (existing) {
      return existing.db;
    }

    // Ensure database directory exists
    const dbPath = join(this.basePath, projectId, "pgdata");
    await this.ensureDirectory(dbPath);

    // Create PGLite instance
    const pglite = new PGlite(dbPath);

    // Create Drizzle instance
    const db = drizzle(pglite, { schema: projectSchema });

    // Store instance
    this.instances.set(projectId, { pglite, db });

    return db;
  }

  /**
   * Initialize a project database with schema
   */
  async initializeDatabase(projectId: string): Promise<void> {
    const db = await this.getInstance(projectId);

    // Check if tables already exist by querying information_schema
    const tables = await db.execute(sql`
      SELECT table_name 
      FROM information_schema.tables 
      WHERE table_schema = 'public' 
      AND table_name IN ('user', 'account', 'authSettings', 'database', 'collection', 'document')
    `);

    // If tables already exist, skip initialization
    if (tables.rows.length > 0) {
      return;
    }

    // Create enum type if it doesn't exist
    await db.execute(sql`
      DO $$ BEGIN
        CREATE TYPE auth_provider AS ENUM ('email', 'google', 'github', 'facebook', 'apple');
      EXCEPTION
        WHEN duplicate_object THEN null;
      END $$;
    `);

    // Create users table
    await db.execute(sql`
      CREATE TABLE IF NOT EXISTS "user" (
        "id" text PRIMARY KEY NOT NULL,
        "name" text NOT NULL,
        "email" text NOT NULL UNIQUE,
        "emailVerified" boolean DEFAULT false NOT NULL,
        "image" text,
        "createdAt" timestamp DEFAULT now() NOT NULL,
        "updatedAt" timestamp DEFAULT now() NOT NULL
      )
    `);

    // Create accounts table
    await db.execute(sql`
      CREATE TABLE IF NOT EXISTS "account" (
        "id" text PRIMARY KEY NOT NULL,
        "userId" text NOT NULL,
        "accountId" text NOT NULL,
        "providerId" text NOT NULL,
        "accessToken" text,
        "refreshToken" text,
        "accessTokenExpiresAt" timestamp,
        "refreshTokenExpiresAt" timestamp,
        "scope" text,
        "idToken" text,
        "password" text,
        "createdAt" timestamp DEFAULT now() NOT NULL,
        "updatedAt" timestamp DEFAULT now() NOT NULL,
        CONSTRAINT "account_userId_user_id_fk" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE cascade
      )
    `);

    // Create authSettings table
    await db.execute(sql`
      CREATE TABLE IF NOT EXISTS "authSettings" (
        "id" text PRIMARY KEY NOT NULL,
        "providers" jsonb DEFAULT '["email"]'::jsonb NOT NULL,
        "emailAndPassword" jsonb,
        "socialProviders" jsonb,
        "createdAt" timestamp DEFAULT now() NOT NULL,
        "updatedAt" timestamp DEFAULT now() NOT NULL
      )
    `);

    // Create databases table (user-created databases within the project)
    await db.execute(sql`
      CREATE TABLE IF NOT EXISTS "database" (
        "databaseId" text PRIMARY KEY NOT NULL,
        "name" text NOT NULL,
        "createdAt" timestamp DEFAULT now() NOT NULL,
        "updatedAt" timestamp DEFAULT now() NOT NULL
      )
    `);

    // Create collections table
    await db.execute(sql`
      CREATE TABLE IF NOT EXISTS "collection" (
        "collectionId" text PRIMARY KEY NOT NULL,
        "databaseId" text NOT NULL,
        "name" text NOT NULL,
        "path" text NOT NULL UNIQUE,
        "parentDocumentId" text,
        "parentPath" text,
        "createdAt" timestamp DEFAULT now() NOT NULL,
        "updatedAt" timestamp DEFAULT now() NOT NULL,
        CONSTRAINT "collection_databaseId_database_databaseId_fk" FOREIGN KEY ("databaseId") REFERENCES "database"("databaseId") ON DELETE cascade
      )
    `);

    // Create documents table
    await db.execute(sql`
      CREATE TABLE IF NOT EXISTS "document" (
        "documentId" text PRIMARY KEY NOT NULL,
        "collectionId" text NOT NULL,
        "path" text NOT NULL UNIQUE,
        "data" jsonb NOT NULL,
        "createdAt" timestamp DEFAULT now() NOT NULL,
        "updatedAt" timestamp DEFAULT now() NOT NULL,
        CONSTRAINT "document_collectionId_collection_collectionId_fk" FOREIGN KEY ("collectionId") REFERENCES "collection"("collectionId") ON DELETE cascade
      )
    `);

    // Create indexes for better performance
    await db.execute(sql`
      CREATE INDEX IF NOT EXISTS "account_userId_idx" ON "account"("userId")
    `);
    await db.execute(sql`
      CREATE INDEX IF NOT EXISTS "collection_databaseId_idx" ON "collection"("databaseId")
    `);
    await db.execute(sql`
      CREATE INDEX IF NOT EXISTS "document_collectionId_idx" ON "document"("collectionId")
    `);
    await db.execute(sql`
      CREATE INDEX IF NOT EXISTS "collection_parentDocumentId_idx" ON "collection"("parentDocumentId")
    `);

    // Insert default auth settings
    await db.execute(sql`
      INSERT INTO "authSettings" ("id", "providers", "emailAndPassword")
      VALUES ('default', '["email"]'::jsonb, '{"enabled": true, "requireEmailVerification": false}'::jsonb)
      ON CONFLICT ("id") DO NOTHING
    `);
  }

  /**
   * Close a database instance
   */
  async closeInstance(projectId: string): Promise<void> {
    const instance = this.instances.get(projectId);
    if (instance) {
      await instance.pglite.close();
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
    return join(this.basePath, projectId, "pgdata");
  }

  /**
   * Ensure directory exists
   */
  private async ensureDirectory(path: string): Promise<void> {
    if (!existsSync(path)) {
      await mkdir(path, { recursive: true });
    }
  }
}
