import { PGlite } from "@electric-sql/pglite";
import { drizzle } from "drizzle-orm/pglite";
import * as schema from "./schema.ts";
import { join } from "path";

// Initialize PGLite database
// Using persistent storage
const pglite = new PGlite(join(import.meta.dir, "../../.data"));

// Create Drizzle client
export const db = drizzle(pglite, { schema });

// Export the PGLite instance for direct access if needed
export { pglite };

// Export schema for use in other files
export * from "./schema.ts";
