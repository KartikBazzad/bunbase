// client.ts
import { treaty } from "@elysiajs/eden";
import type { AppType } from "./server";

export const apiClient = treaty<AppType>("http://localhost:3000", {
  fetch: {
    credentials: "include", // Include cookies for authentication
  },
  headers: {
    "Content-Type": "application/json",
  },
});
