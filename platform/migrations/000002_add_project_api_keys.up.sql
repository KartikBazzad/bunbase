ALTER TABLE projects ADD COLUMN public_api_key VARCHAR(255) UNIQUE;
CREATE INDEX idx_projects_public_api_key ON projects(public_api_key);
