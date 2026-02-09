-- Instance admins: root admins for self-hosted deployments. Only these users can create projects when DEPLOYMENT_MODE=self_hosted.
-- We assume `users` table exists (BunAuth).
CREATE TABLE IF NOT EXISTS instance_admins (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_instance_admins_user_id ON instance_admins(user_id);
