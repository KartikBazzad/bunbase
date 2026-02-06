DROP INDEX IF EXISTS idx_projects_function_subdomain;
ALTER TABLE projects DROP COLUMN IF EXISTS function_subdomain;
