# Documentation Website Requirements

A world-class platform needs world-class documentation. The **BunBase Docs** will be the primary resource for learning, ensuring a developer experience that rivals Vercel or Stripe.

## Goals
-   **Premium Architecture**: Fast, searchable, and beautiful.
-   **Interactive**: Integrated playground (Monaco) to try snippets.
-   **Versioning**: Support for API version toggling.
-   **Auto-generated**: API References generated from Go/TypeScript structs.

## Tech Stack
-   **Framework**: **Fumadocs** (Next.js) or **Starlight** (Astro).
    -   *Decision*: **Fumadocs** (Next.js). Matches our React expertise (Web Console) and offers robust MDX support with "premium" default styling.
-   **Styling**: TailwindCSS.
-   **Search**: Orama (Client-side, fast) or Algolia.
-   **Deployment**: Vercel / Cloudflare Pages / BunBase (Self-hosted).

## Site Structure

### 1. Landing / Introduction
-   "What is BunBase?"
-   "Quickstart": 5-minute tutorial (Install CLI -> Deploy Function).
-   "Architecture Concepts": Explaining BunAuth, Bundoc, etc.

### 2. Guides (The "How-To")
-   **Authentication**: "Adding Social Login", "Protecting Routes".
-   **Database**: "Designing Schemas", "Realtime Subscriptions".
-   **Functions**: "Using npm Packages", "Isolating Environments", "Cron Jobs".
-   **Storage**: "Handling Image Uploads".

### 3. API Reference (The "What")
-   **Client SDK (`bunbase-js`)**: Auto-generated TypeDoc.
    -   `auth.login()`, `store().get()`.
-   **Server SDK (`bunbase-admin`)**.
-   **Platform API**: OpenAPI/Swagger UI embed.
-   **CLI Reference**: Command flags and usage.

### 4. Interactive Playground (Future)
-   Embedded Monaco editor to run simple Bundoc queries against a demo database.

## Content Strategy
-   **Diataxis Framework**: Structure content into Tutorials, How-to Guides, Reference, and Explanation.
-   **Code Grouping**: Show examples in JavaScript, TypeScript, Go, and cURL side-by-side.

## Directory
```
docs/
  content/
    docs/
      getting-started/
      guides/
      api-reference/
  public/
  src/
  package.json
```
