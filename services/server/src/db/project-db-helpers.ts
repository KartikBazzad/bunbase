import { ProjectDatabaseManager } from "./project-db-manager";
import { type BunSQLiteDatabase } from "drizzle-orm/bun-sqlite";
import * as projectSchema from "./project-schema";
import { initializeProjectDatabase } from "./project-db-init";

const manager = ProjectDatabaseManager.getInstance();

/**
 * Get the Drizzle database instance for a project
 * Returns the database instance if it exists, otherwise creates it
 */
export async function getProjectDb(
  projectId: string,
): Promise<BunSQLiteDatabase<typeof projectSchema>> {
  return manager.getInstance(projectId);
}

/**
 * Ensure a project database exists and is initialized
 * Creates and initializes the database if it doesn't exist
 */
export async function ensureProjectDb(
  projectId: string,
): Promise<BunSQLiteDatabase<typeof projectSchema>> {
  // Check if instance exists
  if (!manager.hasInstance(projectId)) {
    // Initialize the database (creates instance and sets up schema)
    await initializeProjectDatabase(projectId);
  }

  // Return the instance
  return manager.getInstance(projectId);
}

/**
 * Check if a project database instance is currently loaded
 */
export function isProjectDbLoaded(projectId: string): boolean {
  return manager.hasInstance(projectId);
}
