# Deploy BunBase in Cloud Mode

In **Cloud mode**, anyone can sign up and create projects. This is the default deployment.

## Prerequisites

- **Docker** and **Docker Compose**
- (Optional) **Go 1.22+** and **Bun 1.1+** if you want the CLI

## 1. Deploy the stack

From the repository root:

```bash
docker compose up -d
```

Or use the helper script:

```bash
./scripts/deploy-cloud.sh
```

Wait for services to be healthy (Postgres, then bun-auth, then platform, etc.). Check:

```bash
docker compose ps
curl -s http://localhost:3001/health
# => {"status":"ok"}
```

## 2. Open the dashboard

- **URL**: http://localhost (Traefik on port 80 serves the dashboard and forwards `/api` to the Platform API.)
- If you only run the platform (no Traefik): use http://localhost:3001 and run platform-web separately with `VITE_API_URL=http://localhost:3001/v1`.

## 3. Create an account and project

1. Open http://localhost (or your Traefik URL).
2. Click **Sign up** and register (email, password, name).
3. After login, click **Create Project** and name your project.
4. Use the project to deploy functions, use the database, and manage API keys.

## 4. (Optional) Use the CLI

Build and use the BunBase CLI against the same API:

```bash
cd platform && go build -o bunbase ./cmd/cli
./bunbase login --api-url http://localhost:3001
# Then: ./bunbase projects list
```

If the dashboard is at http://localhost, the API is at http://localhost/api (same host). For CLI, use the platform port directly, e.g. `--api-url http://localhost:3001`.

## Cloud vs self‑hosted

- **Cloud** (this guide): `PLATFORM_DEPLOYMENT_MODE` is unset or `cloud`. Sign up is open; any user can create projects.
- **Self‑hosted**: Set `PLATFORM_DEPLOYMENT_MODE=self_hosted` in the platform service. Then only the first user (created via **Setup**) can create projects; sign up is disabled after setup. See [Self-hosted and Casbin plan](plans/self-hosted-deployment-and-casbin.md).

## Troubleshooting

- **Dashboard loads but API calls fail**: Ensure you use the same host as the dashboard (e.g. http://localhost) so `/api` is routed by Traefik. If you use http://localhost:5173 (Vite dev), set `VITE_API_URL=http://localhost:3001/v1` in platform-web.
- **403 Forbidden on project pages** (e.g. `/api/projects/.../functions`, config, database): This can happen if **platform migrations did not run**. The platform needs the `project_members` table (and `projects.owner_id`) so project access checks work. See **Verify migrations** below.
- **"relation instance_admins does not exist"** or **"relation project_members does not exist"**: Platform migrations did not run. Ensure the platform started after Postgres was healthy, then verify migrations (see below).
- **Health check fails**: Run `docker compose logs platform` and fix any DB or dependency errors.

### Verify migrations

Platform runs migrations automatically on startup. If the platform started before Postgres was ready, migrations may have failed (the server would have exited). After the stack is up, you can confirm migrations ran:

```bash
./scripts/verify-migrations.sh
```

Or manually:

```bash
# Check migration version (platform uses golang-migrate; table is schema_migrations)
docker compose exec postgres psql -U bunadmin -d bunbase_system -c "SELECT version, dirty FROM schema_migrations;"
```

You should see a single row with the latest version (e.g. `4` for `000004_add_instance_admins`) and `dirty = false`. If the table is missing or the version is old, restart the platform so it runs migrations again (Postgres must be healthy first):

```bash
docker compose up -d postgres
# Wait a few seconds for Postgres to be healthy, then:
docker compose up -d platform
docker compose logs platform
```

To confirm the `project_members` table exists:

```bash
docker compose exec postgres psql -U bunadmin -d bunbase_system -c "\dt project_members"
```
