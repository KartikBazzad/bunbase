import { existsSync, rmSync } from "fs";
import { ProjectDatabaseManager } from "./project-db-manager";
import { join } from "path";

const manager = ProjectDatabaseManager.getInstance();

/**
 * Initialize a new project database
 * Creates the database directory, Bun.SQLite instance, and sets up the schema
 */
export async function initializeProjectDatabase(
  projectId: string,
): Promise<void> {
  try {
    await manager.createInstance(projectId);

    await manager.initializeDatabase(projectId);
  } catch (error) {
    console.error("Failed to initialize project database", error);
    const errorMessage =
      error instanceof Error
        ? error.message
        : typeof error === "string"
          ? error
          : "Unknown error";
    const errorDetails =
      error instanceof Error && error.stack ? `\nStack: ${error.stack}` : "";
    throw new Error(
      `Failed to initialize project database: ${errorMessage}${errorDetails}`,
    );
  }
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

  // Delete the database file and project directory
  const dbPath = manager.getDatabasePath(projectId);
  const projectDir = join(dbPath, "..");

  // Delete the database file if it exists
  if (existsSync(dbPath)) {
    rmSync(dbPath, { force: true });
  }

  // Delete the project directory if it's empty or contains only the database file
  if (existsSync(projectDir)) {
    try {
      rmSync(projectDir, { recursive: true, force: true });
    } catch (error) {
      // Ignore errors if directory is not empty or already deleted
    }
  }
}

/**
 * Check if a project database exists
 */
export async function projectDatabaseExists(
  projectId: string,
): Promise<boolean> {
  const dbPath = manager.getDatabasePath(projectId);
  const dbFile = Bun.file(dbPath);
  return await dbFile.exists();
}
