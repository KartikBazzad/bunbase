import { migrate } from "drizzle-orm/bun-sqlite/migrator";
import { drizzle } from "drizzle-orm/bun-sqlite";
import { Database } from "bun:sqlite";
import { existsSync, mkdirSync } from "fs";

if (!existsSync("./.data/database.db")) {
  mkdirSync("./.data", { recursive: true });
}

const sqlite = new Database("./.data/database.db", {
  readwrite: true,
  create: true,
});
const db = drizzle(sqlite);

console.log("Migrating database...");

migrate(db as any, { migrationsFolder: "./src/db/migrations" });
