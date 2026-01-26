import { Database } from "bun:sqlite";
import { drizzle } from "drizzle-orm/bun-sqlite";
import * as schema from "./schema.ts";
import { join } from "path";
import { mkdirSync, existsSync } from "fs";

// Initialize Bun.SQLite database
// Using persistent storage
const dbPath = join(import.meta.dir, "../../.data");
const dbFile = join(dbPath, "database.db");

// Ensure directory exists
if (!existsSync(dbPath)) {
  mkdirSync(dbPath, { recursive: true });
}

// Create SQLite database instance
const sqlite = new Database(dbFile);

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

// Create Drizzle client
const db = drizzle(sqlite, { schema });

// Export the SQLite instance for direct access if needed
export { db };

// Export schema for use in other files
export * from "./schema.ts";
