# API path consistency

Single convention so frontend, platform proxy, and bundoc stay in sync.

## Public API (Console / Frontend)

All paths use **singular** `database` (one database per project). Base:

```
/api/projects/:id/database
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

| Layer | Base path | Note |
|-------|-----------|------|
| Frontend / Platform routes | `/api/projects/:id/database/...` | Singular `database` |
| Platform → Bundoc (path arg) | `/databases/default/...` | Constant: `bundoc.BundocDBPath` |
| Bundoc server | `/v1/projects/:id/databases/default/...` | Client prepends `/v1/projects/:id` |

Do not add ad-hoc path strings; use `bundoc.BundocDBPath` (and the patterns above) when adding new database routes.
