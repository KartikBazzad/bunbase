import { ProjectDatabaseManager } from "./project-db-manager";
import { rm } from "fs/promises";
import { existsSync } from "fs";
import { join } from "path";

const manager = ProjectDatabaseManager.getInstance();

/**
 * Initialize a new project database
 * Creates the database directory, PGLite instance, and sets up the schema
 */
export async function initializeProjectDatabase(projectId: string): Promise<void> {
  // Create instance and initialize schema
  await manager.createInstance(projectId);
  await manager.initializeDatabase(projectId);
}

/**
 * Delete a project database
 * Closes the instance and removes the database directory
 */
export async function deleteProjectDatabase(projectId: string): Promise<void> {
  // Close the instance if it's open
  if (manager.hasInstance(projectId)) {
    await manager.closeInstance(projectId);
  }

  // Delete the database directory
  const dbPath = manager.getDatabasePath(projectId);
  const projectDir = join(dbPath, "..");

  if (existsSync(projectDir)) {
    await rm(projectDir, { recursive: true, force: true });
  }
}

/**
 * Check if a project database exists
 */
export function projectDatabaseExists(projectId: string): boolean {
  const dbPath = manager.getDatabasePath(projectId);
  return existsSync(dbPath);
}
