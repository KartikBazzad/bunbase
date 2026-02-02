# Web Console Requirements (`platform-web`)

The **Web Console** is the face of BunBase. It must be a premium, high-performance Single Page Application (SPA) that allows developers to manage their projects, data, and functions.

## Design Philosophy
-   **Aesthetic**: "Premium Developer Tool". Dark mode default, high contrast, subtle gradients, glassmorphism accents.
-   **Performance**: Instant transitions (Optimistic UI), Real-time data updates.
-   **Density**: Information-dense but not cluttered.

## Tech Stack
-   **Framework**: React (Vite).
-   **Language**: TypeScript.
-   **Styling**: TailwindCSS (v4).
-   **State**: TanStack Query (Server state), Zustand (Global UI state).
-   **Icons**: Lucide React.
-   **Editor**: Monaco Editor (for online Function editing).
-   **Charts**: Recharts (Monitoring).

## Core Modules

### 1. Authentication Flow
-   **Login/Register**: Clean, centered card layout.
-   **Social Auth**: GitHub/Google buttons (Future).
-   **Organization Switcher**: Dropdown in sidebar to switch context.

### 2. Project Dashboard (Home)
-   **Overview Cards**:
    -   API Requests (24h sparkline).
    -   Storage Usage.
    -   Active Functions.
-   **Quick Actions**: "New Function", "Browse Data".

### 3. Data Browser (Bundoc Interface)
-   **Collection List**: Sidebar showing Collections/Stores.
-   **Document List**: Table/List view of documents with infinite scroll.
-   **Document Editor**: JSON editor with validation for creating/editing records.
-   **Real-time**: Table should update automatically when data changes.

### 4. Functions Manager
-   **List View**: Status indicators (Deployed, Failed), last invoked time.
-   **Detail View**:
    -   **Metrics**: Invocations, Latency, Error Rate graphs.
    -   **Logs**: Live-tailing logs console (WebSocket connected to `buncast`).
    -   **Secrets**: Environment variable manager (masked inputs).
-   **Online Editor**: Monaco instance to edit/deploy simple functions directly from browser.

### 5. Settings
-   **API Keys**: Generate/Revoke public and private keys.
-   **Usage & Billing**: Stripe integration view.
-   **Team Members**: Invite via email, Role assignment.

## UX Requirements
1.  **Optimistic Updates**: When deleting a doc, remove it from UI immediately, revert if API fails.
2.  **Command Palette**: `Cmd+K` to navigate anywhere (Projects, Settings, Docs).
3.  **Toasts**: specialized notifications for all async actions.
4.  **Loading States**: Skeleton screens instead of spinners where possible.

## Directory Structure
```
src/
  components/
    ui/          # Primitives (Button, Input)
    layout/      # Sidebar, Header
    features/    # Domain specific components
  hooks/         # Custom React hooks
  lib/           # API client,Utils
  pages/         # Route pages
  stores/        # Zustand stores
```
