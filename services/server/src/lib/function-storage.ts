/**
 * Function Storage Management
 * Handles filesystem storage for function code with versioning
 */

import { mkdir, writeFile, readFile, rm, stat } from "fs/promises";
import { existsSync } from "fs";
import { join, dirname } from "path";
import { createHash } from "crypto";

const FUNCTIONS_BASE_DIR = join(import.meta.dir, "../../functions");

/**
 * Get the base directory for a project's functions
 */
function getProjectFunctionsDir(projectId: string): string {
  return join(FUNCTIONS_BASE_DIR, projectId);
}

/**
 * Get the directory for a specific function
 */
function getFunctionDir(projectId: string, functionId: string): string {
  return join(getProjectFunctionsDir(projectId), functionId);
}

/**
 * Get the directory for a specific function version
 */
function getVersionDir(
  projectId: string,
  functionId: string,
  version: string,
): string {
  return join(getFunctionDir(projectId, functionId), "versions", version);
}


/**
 * Calculate hash of code content
 */
export function calculateCodeHash(code: string): string {
  return createHash("sha256").update(code).digest("hex");
}

/**
 * Ensure the functions base directory exists
 */
async function ensureBaseDir(): Promise<void> {
  if (!existsSync(FUNCTIONS_BASE_DIR)) {
    await mkdir(FUNCTIONS_BASE_DIR, { recursive: true });
  }
}

/**
 * Store function code for a version
 */
export async function storeFunctionCode(
  projectId: string,
  functionId: string,
  version: string,
  code: string,
): Promise<{ codePath: string; codeHash: string }> {
  await ensureBaseDir();

  const versionDir = getVersionDir(projectId, functionId, version);
  await mkdir(versionDir, { recursive: true });

  const codePath = join(versionDir, "code.ts");
  const codeHash = calculateCodeHash(code);

  await writeFile(codePath, code, "utf-8");

  return { codePath, codeHash };
}

/**
 * Read function code for a version
 */
export async function readFunctionCode(
  projectId: string,
  functionId: string,
  version: string,
): Promise<string> {
  const versionDir = getVersionDir(projectId, functionId, version);
  const codePath = join(versionDir, "code.ts");

  if (!existsSync(codePath)) {
    throw new Error(`Function code not found for version ${version}`);
  }

  return await readFile(codePath, "utf-8");
}

/**
 * Get the code path for a version
 */
export function getFunctionCodePath(
  projectId: string,
  functionId: string,
  version: string,
): string {
  return join(getVersionDir(projectId, functionId, version), "code.ts");
}

/**
 * Get the code path for a version (used when we have version string)
 */
export function getCodePathForVersion(
  projectId: string,
  functionId: string,
  version: string,
): string {
  return getFunctionCodePath(projectId, functionId, version);
}

/**
 * Delete a function version
 */
export async function deleteFunctionVersion(
  projectId: string,
  functionId: string,
  version: string,
): Promise<void> {
  const versionDir = getVersionDir(projectId, functionId, version);
  if (existsSync(versionDir)) {
    await rm(versionDir, { recursive: true });
  }
}

/**
 * Delete all function code for a function
 */
export async function deleteFunctionCode(
  projectId: string,
  functionId: string,
): Promise<void> {
  const functionDir = getFunctionDir(projectId, functionId);
  if (existsSync(functionDir)) {
    await rm(functionDir, { recursive: true });
  }
}

/**
 * List all versions for a function
 */
export async function listFunctionVersions(
  projectId: string,
  functionId: string,
): Promise<string[]> {
  const versionsDir = join(getFunctionDir(projectId, functionId), "versions");
  if (!existsSync(versionsDir)) {
    return [];
  }

  // Read directory and return version names
  const { readdir } = await import("fs/promises");
  const entries = await readdir(versionsDir, { withFileTypes: true });
  return entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name);
}

/**
 * Check if a version exists
 */
export async function versionExists(
  projectId: string,
  functionId: string,
  version: string,
): Promise<boolean> {
  const versionDir = getVersionDir(projectId, functionId, version);
  const codePath = join(versionDir, "code.ts");
  return existsSync(codePath);
}
