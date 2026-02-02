# BunBase SDK Requirements

We will provide two official SDKs to support client-side and server-side development.

## 1. Client SDK (`bunbase-js`)
**Target**: Browser, React Native, Electron.
**Auth**: Uses Public API Key (identifies project).
**Role**: Access user-centric data (Bundoc) and public functions.

### Modules

#### `auth`
-   `login(email, password)`
-   `register(email, password, name)`
-   `logout()`
-   `watchSession(callback)`: Real-time session listener.

#### `data` (Bundoc)
-   `store(name).get(id)`
-   `store(name).put(id, data)`: Overwrite or Create.
-   `store(name).patch(id, data)`: Partial update.
-   `store(name).watch(id, callback)`: Real-time updates via SSE.

#### `functions`
-   `call(functionName, payload)`: Calls HTTP Gateway.

#### `storage` (MinIO)
-   `bucket(name).upload(file)`
-   `bucket(name).downloadUrl(path)`

### Usage Example
```javascript
import { Client } from 'bunbase-js';

const app = new Client({ apiKey: "pk_..." });

// Auth
await app.auth.login("user@example.com", "pass");

// Data
const doc = await app.data.store("todos").get("1");
console.log(doc);

// Realtime
app.data.store("todos").watch("1", (updated) => {
  console.log("New data:", updated);
});
```

---

## 2. Admin SDK (`bunbase-admin`)
**Target**: Node.js, Bun server-side environments.
**Auth**: Uses Service Account Secret (Master access).
**Role**: Privileged backend operations.

### Modules

#### `auth`
-   `verifyIdToken(token)`: Middleware helper.
-   `getUser(uid)`
-   `createUser(properties)`
-   `deleteUser(uid)`
-   `createCustomToken(uid, claims)`

#### `doc` (Bundoc)
-   Same API as Client SDK, but bypasses security rules (future) / has full access.

#### `functions`
-   `deploy(name, path)`: CI/CD deployment helper.

### Usage Example
```javascript
import { initializeApp, cert } from 'bunbase-admin';
import { getAuth } from 'bunbase-admin/auth';

const app = initializeApp({
  credential: cert("./service-account.json")
});

const decoded = await getAuth(app).verifyIdToken(token);
console.log("Verified User:", decoded.uid);
```

## Implementation Strategy
-   **Monorepo**: Create `packages/bunbase-js` and `packages/bunbase-admin` in the main repo.
-   **Bundling**: Use `bun build` to emit ESM and CJS formats.
-   **Types**: Full TypeScript support is mandatory.
