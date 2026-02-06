ALTER TABLE projects ADD COLUMN IF NOT EXISTS function_subdomain VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_projects_function_subdomain ON projects(function_subdomain);

