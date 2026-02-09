# BunBase Demo App

A small web app that showcases **documents**, **authentication**, and **functions** using the [BunBase TypeScript SDK](../../bunbase-js) and documents CLI usage for deploying functions.

## Prerequisites

- Node.js 18+ or Bun
- Platform and services running (e.g. via [docker-compose](../../docker-compose.yml) or locally: platform on port 3001, bundoc, auth, functions gateway)

## Setup

1. **Install dependencies**

   ```bash
   cd demo-app
   npm install
   ```

   The app depends on the local SDK (`bunbase-js`). Ensure the SDK is built:

   ```bash
   cd ../bunbase-js && npm run build && cd ../demo-app
   ```

2. **Environment (optional)**

   Create a `.env` file to pre-fill the app:

   ```env
   VITE_BUNBASE_URL=http://localhost:3001
   VITE_PROJECT_ID=your-project-id
   VITE_API_KEY=your-project-api-key
   ```

   Or set **Project API key** and **Project ID** in the app under **Settings** (see below).

3. **Run the demo**

   ```bash
   npm run dev
   ```

   Opens at `http://localhost:5174` (or the next free port). If the platform runs on another host, set `VITE_BUNBASE_URL` and use the proxy in `vite.config.ts` or point the app at the platform URL.

## Getting Project API key and Project ID

- **Project API key**: In the [BunBase dashboard](http://localhost:5173) (platform-web), open a project and go to **Settings**. Copy the **API key** and paste it in the demo app **Settings** as “API token”.
- **Project ID**: From the dashboard, the project ID is in the URL (`/projects/<id>/...`) or in the projects list. Paste it in **Settings** as “Project ID”.

The demo uses the project API key with the SDK (sent as `X-Bunbase-Client-Key`) to access that project's database and functions.

## Using the CLI

The BunBase CLI is documented in [docs/users/cli-guide.md](../docs/users/cli-guide.md). When the CLI binary is available (e.g. built from `platform/cmd/cli`), you can deploy a function and then invoke it from this demo.

### Deploy a function

1. Log in and select a project:

   ```bash
   bunbase auth login
   bunbase projects list
   bunbase projects use <project-id>
   ```

2. Deploy the example function:

   ```bash
   bunbase deploy ../../functions/examples/hello-world.ts --name hello-world --runtime bun --handler default
   ```

   Or use the helper script (when CLI is on your PATH):

   ```bash
   ./scripts/deploy-demo-function.sh
   ```

3. In the demo app, open **Functions**, set the function name to `hello-world`, optionally set the body to `{"name": "World"}`, and click **Invoke**.

### List functions

```bash
bunbase functions list
```

## App features

- **Home**: Short intro and links to Documents, References, Functions, Settings.
- **Documents**: CRUD on a `tasks` collection using the SDK (`client.db.collection('tasks').list()`, `.create()`, `.update()`, `.delete()`). Create a task, edit, and delete from the list.
- **References**: Cross-collection references demo: create users and posts (with `author_id` → users), see 409 on invalid reference, and try restrict / set_null / cascade when deleting a user.
- **Functions**: Invoke a function by name with optional JSON body via `client.functions.invoke(name, body)`.
- **Settings**: Set base URL, Project API key (from dashboard Project → Settings), and Project ID. Stored in `localStorage`.

## See also

- [BunBase TypeScript SDK](../../bunbase-js)
- [CLI guide](../docs/users/cli-guide.md)
- [Getting started](../docs/users/getting-started.md)
- [Writing functions](../docs/users/writing-functions.md)
