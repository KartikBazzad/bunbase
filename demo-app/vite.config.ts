import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      // Use SDK source so Vite can bundle ESM (dist is CJS and breaks Rollup)
      "bunbase-js": path.resolve(__dirname, "../bunbase-js/src/index.ts"),
    },
  },
  server: {
    port: 5174,
    proxy: {
      "/v1": {
        target: "http://localhost:3001",
        changeOrigin: true,
        cookieDomainRewrite: "localhost",
        secure: false,
      },
      "/api": {
        target: "http://localhost:3001",
        changeOrigin: true,
        cookieDomainRewrite: "localhost",
        secure: false,
      },
    },
  },
});
