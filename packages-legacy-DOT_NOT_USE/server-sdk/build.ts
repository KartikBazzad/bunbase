#!/usr/bin/env bun
import { existsSync } from "fs";
import { rm } from "fs/promises";
import path from "path";

const formatFileSize = (bytes: number): string => {
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`;
};

console.log("\n🚀 Starting server-sdk build process...\n");

const outdir = path.join(process.cwd(), "dist");

// Clean previous build
if (existsSync(outdir)) {
  console.log(`🗑️  Cleaning previous build at ${outdir}`);
  await rm(outdir, { recursive: true, force: true });
}

const start = performance.now();

// Build ESM format
console.log("📦 Building ESM format...");
const esmResult = await Bun.build({
  entrypoints: ["./src/index.ts"],
  outdir,
  format: "esm",
  target: "bun",
  minify: false,
  sourcemap: "linked",
  naming: {
    entry: "[dir]/[name].mjs",
  },
  external: ["zod"], // External dependencies
});

if (!esmResult.success) {
  console.error("❌ ESM build failed:");
  esmResult.logs.forEach((log) => console.error(log));
  process.exit(1);
}

// Build CJS format
console.log("📦 Building CJS format...");
const cjsResult = await Bun.build({
  entrypoints: ["./src/index.ts"],
  outdir,
  format: "cjs",
  target: "bun",
  minify: false,
  sourcemap: "linked",
  naming: {
    entry: "[dir]/[name].js",
  },
  external: ["zod"], // External dependencies
});

if (!cjsResult.success) {
  console.error("❌ CJS build failed:");
  cjsResult.logs.forEach((log) => console.error(log));
  process.exit(1);
}

// Generate type definitions using tsc
console.log("📝 Generating type definitions...");
const tscProcess = Bun.spawn({
  cmd: ["bunx", "tsc", "--emitDeclarationOnly", "--declaration", "--declarationMap"],
  cwd: process.cwd(),
  stdout: "inherit",
  stderr: "inherit",
});

const tscExitCode = await tscProcess.exited;
if (tscExitCode !== 0) {
  console.error("❌ Type definition generation failed");
  process.exit(1);
}

const end = performance.now();

// Display build results
const allOutputs = [...esmResult.outputs, ...cjsResult.outputs];
const outputTable = allOutputs.map((output) => ({
  File: path.relative(process.cwd(), output.path),
  Type: output.kind,
  Size: formatFileSize(output.size),
}));

console.table(outputTable);
const buildTime = ((end - start) / 1000).toFixed(2);

console.log(`\n✅ Build completed in ${buildTime}s\n`);
