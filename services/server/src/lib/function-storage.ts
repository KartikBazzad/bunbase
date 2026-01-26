/**
 * Function Storage Management
 * Handles filesystem storage for function code with versioning
 */

import { join } from "path";

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
export async function calculateCodeHash(code: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(code);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Ensure the functions base directory exists
 */
async function ensureBaseDir(): Promise<void> {
  try {
    // Try to read the directory - if it fails, create it
    await Bun.readdir(FUNCTIONS_BASE_DIR);
  } catch {
    // Directory doesn't exist, create it by writing a file
    // This will create the directory structure
    await Bun.write(join(FUNCTIONS_BASE_DIR, ".keep"), "");
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
  // Ensure directory exists by trying to read it, or create it
  try {
    await Bun.readdir(versionDir);
  } catch {
    // Directory doesn't exist, create it by writing a file
    await Bun.write(join(versionDir, ".keep"), "");
  }

  const codePath = join(versionDir, "code.ts");
  const codeHash = await calculateCodeHash(code);

  await Bun.write(codePath, code);

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
  const file = Bun.file(codePath);

  if (!(await file.exists())) {
    throw new Error(`Function code not found for version ${version}`);
  }

  return await file.text();
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
  try {
    // Check if directory exists by trying to read it
    await Bun.readdir(versionDir);
    // Directory exists, delete it using fs/promises (Bun doesn't have recursive delete)
    const { rm } = await import("fs/promises");
    await rm(versionDir, { recursive: true });
  } catch {
    // Directory doesn't exist, nothing to delete
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
  try {
    // Check if directory exists by trying to read it
    await Bun.readdir(functionDir);
    // Directory exists, delete it
    const { rm } = await import("fs/promises");
    await rm(functionDir, { recursive: true });
  } catch {
    // Directory doesn't exist, nothing to delete
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
  
  // Read directory and return version names using Bun.readdir
  try {
    const entries = await Bun.readdir(versionsDir);
    const versionDirs: string[] = [];
    
    for await (const entry of entries) {
      // Check if entry is a directory
      if (entry.isDirectory()) {
        versionDirs.push(entry.name);
      }
    }
    
    return versionDirs;
  } catch {
    return [];
  }
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
  try {
    const file = Bun.file(codePath);
    return await file.exists();
  } catch {
    return false;
  }
}
