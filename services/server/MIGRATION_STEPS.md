# Migration Steps for Default Database Flag

## 1. Generate Migration for Main Database

Run this command to generate a migration for the `isDefault` column addition:

```bash
cd services/server
bunx drizzle-kit generate
```

This will create a new migration file in `src/db/migrations/` that adds the `isDefault` column to the `databases` table.

## 2. Apply Migration

The migration will be automatically applied when the server starts (if you have migration logic), or you can run:

```bash
bunx drizzle-kit migrate
```

## 3. Update Existing Databases

For existing databases in the main `databases` table, you may want to set the first database of each project as default:

```sql
-- This SQL will set the first database (by creation date) of each project as default
UPDATE "database" d1
SET "isDefault" = true
WHERE d1."databaseId" = (
  SELECT d2."databaseId"
  FROM "database" d2
  WHERE d2."projectId" = d1."projectId"
  ORDER BY d2."createdAt" ASC
  LIMIT 1
);
```

## 4. Project Databases

Project databases (the per-project PGLite databases) will automatically get the `isDefault` column when:
- New projects are created (via the updated initialization code)
- Existing project databases are accessed (the column will be added if missing)

If you want to update existing project databases immediately, you'll need to run an ALTER TABLE on each project database, or wait for them to be accessed and the schema will be updated automatically.

## Notes

- The `isDefault` column defaults to `false`, so existing databases won't break
- The API key resolver has a fallback: if no default is found, it uses the first database
- New databases created will automatically be set as default if they're the first one in the project
