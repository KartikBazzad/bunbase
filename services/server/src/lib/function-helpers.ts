/**
 * Function Management Helpers
 * Database operations for functions
 */

import { db } from "../db";
import {
  functions,
  functionVersions,
  functionDeployments,
} from "../db/schema";
import { eq, and, desc } from "drizzle-orm";
import { NotFoundError } from "./errors";
import { logProjectOperation } from "./project-logger-utils";
import {
  storeFunctionCode,
  deleteFunctionCode,
} from "./function-storage";
import { validateFunctionCode } from "./function-validator";
import { nanoid } from "nanoid";

/**
 * Get function by ID with project verification
 */
export async function getFunctionById(
  functionId: string,
  projectId: string,
) {
  const [func] = await db
    .select()
    .from(functions)
    .where(and(eq(functions.id, functionId), eq(functions.projectId, projectId)))
    .limit(1);

  if (!func) {
    throw new NotFoundError("Function", functionId);
  }

  return func;
}

/**
 * Get all functions for a project
 */
export async function getProjectFunctions(projectId: string) {
  return await db
    .select()
    .from(functions)
    .where(eq(functions.projectId, projectId))
    .orderBy(desc(functions.createdAt));
}

/**
 * Create a new function
 */
export async function createFunction(
  projectId: string,
  data: {
    name: string;
    runtime: string;
    handler: string;
    code?: string;
    memory?: number;
    timeout?: number;
  },
) {
  // Validate code if provided
  if (data.code) {
    const validation = await validateFunctionCode(data.code, data.runtime);
    if (!validation.valid) {
      throw new Error(
        `Code validation failed: ${validation.errors.join(", ")}`,
      );
    }
  }

  const functionId = nanoid();

  // Create function record
  const [func] = await db
    .insert(functions)
    .values({
      id: functionId,
      projectId,
      name: data.name,
      runtime: data.runtime || "bun",
      handler: data.handler,
      memory: data.memory || 512,
      timeout: data.timeout || 30,
      status: "draft",
    })
    .returning();

  // Store code if provided
  if (data.code) {
    const version = "1.0.0";
    const { codePath, codeHash } = await storeFunctionCode(
      projectId,
      functionId,
      version,
      data.code,
    );

    // Create version record
    const versionId = nanoid();
    await db.insert(functionVersions).values({
      id: versionId,
      functionId,
      version,
      codeHash,
      codePath,
    });

    // Set as active version
    await db
      .update(functions)
      .set({ activeVersionId: versionId })
      .where(eq(functions.id, functionId));
  }

  // Log function creation
  logProjectOperation(projectId, "function_create", {
    functionId: func.id,
    functionName: func.name,
    runtime: func.runtime,
  });

  return func;
}

/**
 * Update function metadata
 */
export async function updateFunction(
  functionId: string,
  projectId: string,
  data: {
    name?: string;
    runtime?: string;
    handler?: string;
    code?: string;
    memory?: number;
    timeout?: number;
  },
) {
  const func = await getFunctionById(functionId, projectId);

  // Validate code if provided
  if (data.code) {
    const validation = await validateFunctionCode(
      data.code,
      data.runtime || func.runtime,
    );
    if (!validation.valid) {
      throw new Error(
        `Code validation failed: ${validation.errors.join(", ")}`,
      );
    }
  }

  // Update function record
  const updateData: any = {
    updatedAt: new Date(),
  };
  if (data.name !== undefined) updateData.name = data.name;
  if (data.runtime !== undefined) updateData.runtime = data.runtime;
  if (data.handler !== undefined) updateData.handler = data.handler;
  if (data.memory !== undefined) updateData.memory = data.memory;
  if (data.timeout !== undefined) updateData.timeout = data.timeout;

  const [updated] = await db
    .update(functions)
    .set(updateData)
    .where(eq(functions.id, functionId))
    .returning();

  // Store new code version if provided
  if (data.code) {
    // Get latest version and increment
    const [latestVersion] = await db
      .select()
      .from(functionVersions)
      .where(eq(functionVersions.functionId, functionId))
      .orderBy(desc(functionVersions.createdAt))
      .limit(1);

    let newVersion = "1.0.0";
    if (latestVersion) {
      const parts = latestVersion.version.split(".");
      const minor = parseInt(parts[1] || "0") + 1;
      newVersion = `${parts[0]}.${minor}.0`;
    }

    const { codePath, codeHash } = await storeFunctionCode(
      projectId,
      functionId,
      newVersion,
      data.code,
    );

    // Create version record
    const versionId = nanoid();
    await db.insert(functionVersions).values({
      id: versionId,
      functionId,
      version: newVersion,
      codeHash,
      codePath,
    });

    // Update active version if this is a new deployment
    // (Don't auto-set on code update, only on deploy)
  }

  return updated;
}

/**
 * Delete function
 */
export async function deleteFunction(functionId: string, projectId: string) {
  const func = await getFunctionById(functionId, projectId);

  // Delete function code from filesystem
  await deleteFunctionCode(projectId, functionId);

  // Delete function record (cascade will delete versions, deployments, etc.)
  await db.delete(functions).where(eq(functions.id, functionId));

  // Log function deletion
  logProjectOperation(projectId, "function_delete", {
    functionId: func.id,
    functionName: func.name,
  });

  return func;
}

/**
 * Get function versions
 */
export async function getFunctionVersions(functionId: string, projectId: string) {
  await getFunctionById(functionId, projectId); // Verify function exists

  return await db
    .select()
    .from(functionVersions)
    .where(eq(functionVersions.functionId, functionId))
    .orderBy(desc(functionVersions.createdAt));
}

/**
 * Get active version for a function
 */
export async function getActiveVersion(functionId: string, projectId: string) {
  const func = await getFunctionById(functionId, projectId);

  if (!func.activeVersionId) {
    return null;
  }

  const [version] = await db
    .select()
    .from(functionVersions)
    .where(eq(functionVersions.id, func.activeVersionId))
    .limit(1);

  return version;
}
