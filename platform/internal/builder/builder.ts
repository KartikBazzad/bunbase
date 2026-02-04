import { BuildConfig } from "bun";

// Restrict dangerous modules
const securityPlugin = {
  name: "security-plugin",
  setup(build: any) {
    // Block node builtins that expose system access
    const blockedNodeModules = [
      "fs",
      "fs/promises",
      "path",
      "os",
      "child_process",
      "cluster",
      "dgram",
      "dns",
      "http2",
      "net",
      "tls",
      "process",
      "console", // maybe allow console?
    ];

    // We allow "console" for logging, but others are blocked.
    // Actually, "console" is global, importing it is weird but often done.
    // Let's block "node:fs" explicit imports.

    build.onResolve({ filter: /^node:.*$/ }, (args: any) => {
      const moduleName = args.path.replace("node:", "");
      if (blockedNodeModules.includes(moduleName)) {
        return {
          path: args.path,
          namespace: "blocked",
        };
      }
    });

    build.onResolve({ filter: /^bun.*$/ }, (args: any) => {
      // Allow specific bun modules if needed? No, block all for now.
      // Users should use standard Web APIs or our SDK.
      return {
        path: args.path,
        namespace: "blocked",
      };
    });

    // Handle blocked imports
    build.onLoad({ filter: /.*/, namespace: "blocked" }, (args: any) => {
      return {
        contents: `throw new Error("Module '${args.path}' is restricted in the serverless environment.");`,
        loader: "js",
      };
    });
  },
};

const args = process.argv.slice(2);
if (args.length < 2) {
  console.error("Usage: bun builder.ts <entrypoint> <outdir>");
  process.exit(1);
}

const [entrypoint, outdir] = args;

console.log(`🔨 Building ${entrypoint} to ${outdir}...`);

const result = await Bun.build({
  entrypoints: [entrypoint],
  outdir: outdir,
  target: "bun",
  format: "esm",
  sourcemap: "inline",
  minify: false, // Easier debugging for now
  plugins: [securityPlugin],
});

if (!result.success) {
  console.error("Build failed:");
  for (const message of result.logs) {
    console.error(message);
  }
  process.exit(1);
}

console.log("✅ Build successful");
process.exit(0);
