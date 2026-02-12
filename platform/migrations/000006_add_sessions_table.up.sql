-- Sessions table for platform-managed session tokens
-- Stores JWT tokens server-side and maps session tokens to JWTs
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    jwt_token TEXT NOT NULL,
    session_type VARCHAR(50) NOT NULL CHECK (session_type IN ('platform', 'tenant')),
    user_id UUID,
    project_id UUID,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_session_token ON sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id);

-- Cleanup expired sessions (can be run periodically)
-- DELETE FROM sessions WHERE expires_at < NOW();
