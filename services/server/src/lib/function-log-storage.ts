/**
 * Function Log Storage
 * Stores function logs in SQLite (per-project, append-optimized)
 */

import { Database } from "bun:sqlite";
import { join } from "path";
import { LogEntry } from "./function-log-buffer";

const FUNCTIONS_BASE_DIR = join(import.meta.dir, "../../functions");

/**
 * Get SQLite database path for a project
 */
function getProjectLogDbPath(projectId: string): string {
  const projectDir = join(FUNCTIONS_BASE_DIR, projectId);
  return join(projectDir, "logs.db");
}

/**
 * Get or create SQLite database for project logs
 */
async function getProjectLogDb(projectId: string): Promise<Database> {
  const dbPath = getProjectLogDbPath(projectId);
  const dbDir = join(dbPath, "..");

  // Ensure directory exists using Bun
  const dirFile = Bun.file(dbDir);
  if (!(await dirFile.exists())) {
    // Create directory by writing a keep file
    await Bun.write(join(dbDir, ".keep"), "");
  }

  // Create or open database
  const db = new Database(dbPath);

  // Create table if not exists
  db.exec(`
    CREATE TABLE IF NOT EXISTS function_logs (
      id TEXT PRIMARY KEY,
      functionId TEXT NOT NULL,
      executionId TEXT NOT NULL,
      level TEXT NOT NULL,
      message TEXT NOT NULL,
      metadata TEXT,
      timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_function_logs_functionId 
      ON function_logs(functionId);
    CREATE INDEX IF NOT EXISTS idx_function_logs_executionId 
      ON function_logs(executionId);
    CREATE INDEX IF NOT EXISTS idx_function_logs_timestamp 
      ON function_logs(timestamp);
    CREATE INDEX IF NOT EXISTS idx_function_logs_level 
      ON function_logs(level);
  `);

  return db;
}

/**
 * Flush logs to SQLite storage
 */
export async function flushLogsToStorage(logs: LogEntry[]): Promise<void> {
  if (logs.length === 0) {
    return;
  }

  // Group logs by project (we need to get projectId from functionId)
  // For now, we'll need to query the function to get projectId
  // This is a bit inefficient, but we can optimize later with a cache
  const logsByProject = new Map<string, LogEntry[]>();

  // Get projectId for each functionId
  const { db } = await import("../db");
  const { functions } = await import("../db/schema");
  const { eq, inArray } = await import("drizzle-orm");

  const functionIds = [...new Set(logs.map((log) => log.functionId))];
  const functionRecords = await db
    .select({ id: functions.id, projectId: functions.projectId })
    .from(functions)
    .where(inArray(functions.id, functionIds));

  const functionToProject = new Map(
    functionRecords.map((f) => [f.id, f.projectId]),
  );

  // Group logs by project
  for (const log of logs) {
    const projectId = functionToProject.get(log.functionId);
    if (!projectId) {
      continue; // Skip logs for non-existent functions
    }

    if (!logsByProject.has(projectId)) {
      logsByProject.set(projectId, []);
    }
    logsByProject.get(projectId)!.push(log);
  }

  // Flush logs for each project
  for (const [projectId, projectLogs] of logsByProject.entries()) {
    const db = await getProjectLogDb(projectId);

    // Batch insert
    const insertStmt = db.prepare(`
      INSERT INTO function_logs (id, functionId, executionId, level, message, metadata, timestamp)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `);

    const insertMany = db.transaction((logs: LogEntry[]) => {
      for (const log of logs) {
        insertStmt.run(
          log.id,
          log.functionId,
          log.executionId,
          log.level,
          log.message,
          log.metadata ? JSON.stringify(log.metadata) : null,
          log.timestamp.toISOString(),
        );
      }
    });

    insertMany(projectLogs);
    insertStmt.finalize();
    db.close();
  }
}

/**
 * Get logs for a function
 */
export async function getFunctionLogs(
  projectId: string,
  functionId: string,
  options: {
    limit?: number;
    offset?: number;
    level?: string;
    executionId?: string;
    startDate?: Date;
    endDate?: Date;
  } = {},
): Promise<Array<{
  id: string;
  functionId: string;
  executionId: string;
  level: string;
  message: string;
  metadata: Record<string, any> | null;
  timestamp: Date;
}>> {
  const db = await getProjectLogDb(projectId);

  let query = `SELECT * FROM function_logs WHERE functionId = ?`;
  const params: any[] = [functionId];

  if (options.executionId) {
    query += ` AND executionId = ?`;
    params.push(options.executionId);
  }

  if (options.level) {
    query += ` AND level = ?`;
    params.push(options.level);
  }

  if (options.startDate) {
    query += ` AND timestamp >= ?`;
    params.push(options.startDate.toISOString());
  }

  if (options.endDate) {
    query += ` AND timestamp <= ?`;
    params.push(options.endDate.toISOString());
  }

  query += ` ORDER BY timestamp DESC`;

  if (options.limit) {
    query += ` LIMIT ?`;
    params.push(options.limit);
  }

  if (options.offset) {
    query += ` OFFSET ?`;
    params.push(options.offset);
  }

  const stmt = db.prepare(query);
  const rows = stmt.all(...params) as Array<{
    id: string;
    functionId: string;
    executionId: string;
    level: string;
    message: string;
    metadata: string | null;
    timestamp: string;
  }>;

  stmt.finalize();
  db.close();

  return rows.map((row) => ({
    id: row.id,
    functionId: row.functionId,
    executionId: row.executionId,
    level: row.level,
    message: row.message,
    metadata: row.metadata ? JSON.parse(row.metadata) : null,
    timestamp: new Date(row.timestamp),
  }));
}

/**
 * Clean up old logs (retention policy)
 */
export async function cleanupOldLogs(
  projectId: string,
  retentionDays: number = 30,
): Promise<number> {
  const db = await getProjectLogDb(projectId);
  const cutoffDate = new Date();
  cutoffDate.setDate(cutoffDate.getDate() - retentionDays);

  const stmt = db.prepare(
    `DELETE FROM function_logs WHERE timestamp < ?`,
  );
  const result = stmt.run(cutoffDate.toISOString());
  stmt.finalize();
  db.close();

  return result.changes || 0;
}
