DROP INDEX IF EXISTS idx_projects_public_api_key;
ALTER TABLE projects DROP COLUMN IF EXISTS public_api_key;
