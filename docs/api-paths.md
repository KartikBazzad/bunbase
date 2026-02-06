# API path consistency

Single convention so frontend, platform proxy, and bundoc stay in sync.

## Key-based API (SDK / apps)

For SDK and server-side apps: **only the API key** is required. Project is inferred from the key. No project ID in the path. Auth: `X-Bunbase-Client-Key` header (or query `key=` for SSE).

Base prefix: `/v1` (e.g. platform at `http://localhost:3001/v1`).

| Purpose | Method | Path |
|---------|--------|------|
| Current project + config | GET | `/v1/project` |
| List collections | GET | `/v1/database/collections` |
| Create collection | POST | `/v1/database/collections` |
| Get/update/delete collection | GET / PATCH / DELETE | `/v1/database/collections/:collection` |
| Collection rules | PATCH | `/v1/database/collections/:collection/rules` |
| Indexes | GET / POST / DELETE | `/v1/database/collections/:collection/indexes` (and `.../indexes/:field`) |
| List/create/query documents | GET / POST / POST | `/v1/database/collections/:collection/documents`, `.../documents/query` |
| Get/update/delete document | GET / PUT / PATCH / DELETE | `/v1/database/collections/:collection/documents/:docID` |
| Realtime (SSE) | GET / POST | `/v1/database/collections/:collection/subscribe`, `.../documents/query/subscribe` |
| Invoke function | POST / GET | `/v1/functions/:name/invoke` |
| List functions | GET | `/v1/functions` |
| Tenant auth users | GET / POST | `/v1/auth/users` |
| Tenant auth config | GET / PUT | `/v1/auth/config` |

**SDK** (`bunbase-js`): `createClient(url, apiKey)` — no project ID. Use `client.getProject()` when you need project id/name/config.

## Console API (multi-project)

For the Web Console: **session or Bearer token** and **project ID in path**. Base:

```
/api/projects/:id/database   or   /v1/projects/:id/database
```

| Resource | Method | Path |
|----------|--------|------|
| List collections | GET | `/api/projects/:id/database/collections` |
| Create collection | POST | `/api/projects/:id/database/collections` |
| Get collection (schema, rules) | GET | `/api/projects/:id/database/collections/:collection` |
| Update/Delete collection | PATCH/DELETE | `/api/projects/:id/database/collections/:collection` |
| List documents | GET | `/api/projects/:id/database/collections/:collection/documents` |
| Create document | POST | `/api/projects/:id/database/collections/:collection/documents` |
| Get/Update/Delete document | GET/PUT/DELETE | `/api/projects/:id/database/collections/:collection/documents/:docID` |
| Query documents | POST | `/api/projects/:id/database/collections/:collection/documents/query` |
| Collection rules (update) | PATCH | `/api/projects/:id/database/collections/:collection/rules` |
| Indexes | GET/POST/DELETE | `.../collections/:collection/indexes` (and `.../indexes/:field`) |

**Frontend** (`platform-web`): build URLs like  
`/projects/${projectId}/database/collections/${collection}/documents`  
(API client uses `VITE_API_URL` + these paths; typically `/api` is the platform prefix.)

## Bundoc (internal)

Bundoc uses **plural** `databases` and a `default` segment. Base:

```
/v1/projects/:id/databases/default
```

So the **path suffix** passed from platform to the bundoc client is:

```
/databases/default/collections
/databases/default/collections/:name/documents
/databases/default/collections/:name/documents/:docID
...
```

- **Platform** builds this suffix in `internal/handlers/database.go` using `bundoc.BundocDBPath` (`/databases/default`).
- **Bundoc client** (`ProxyRequest`) builds the full URL: `BaseURL + "/v1/projects/" + projectID + path`.

## Summary

| Layer | Base path | Auth | Note |
|-------|-----------|------|------|
| Key-based (SDK/apps) | `/v1/database/...`, `/v1/functions/...`, `/v1/project`, `/v1/auth/...` | `X-Bunbase-Client-Key` | No project ID in path |
| Console (platform-web) | `/api/projects/:id/database/...` | Session or Bearer | Project ID in path |
| Platform → Bundoc (path arg) | `/databases/default/...` | — | Constant: `bundoc.BundocDBPath` |
| Bundoc server | `/v1/projects/:id/databases/default/...` | — | Client prepends `/v1/projects/:id` |

Do not add ad-hoc path strings; use `bundoc.BundocDBPath` (and the patterns above) when adding new database routes.
