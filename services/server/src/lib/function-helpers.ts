/**
 * Function Helpers
 * Database operations for functions and versions
 */

import { db } from "../db";
import { functions, functionVersions } from "../db/schema";
import { eq, and, desc } from "drizzle-orm";
import { nanoid } from "nanoid";
import { NotFoundError } from "./errors";
import { storeFunctionCode } from "./function-storage";

export interface CreateFunctionData {
  name: string;
  runtime?: string;
  handler: string;
  code?: string;
  memory?: number;
  timeout?: number;
}

export interface UpdateFunctionData {
  name?: string;
  runtime?: string;
  handler?: string;
  code?: string;
  memory?: number;
  timeout?: number;
}

/**
 * Generate a version string using ISO timestamp
 */
function generateVersion(): string {
  return new Date().toISOString();
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
 * Get a function by ID, ensuring it belongs to the project
 */
export async function getFunctionById(functionId: string, projectId: string) {
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
 * Get a function by name, ensuring it belongs to the project
 */
export async function getFunctionByName(projectId: string, name: string) {
  const [func] = await db
    .select()
    .from(functions)
    .where(and(eq(functions.projectId, projectId), eq(functions.name, name)))
    .limit(1);

  if (!func) {
    throw new NotFoundError("Function", name);
  }

  return func;
}

/**
 * Create a new function with optional initial version
 */
export async function createFunction(
  projectId: string,
  data: CreateFunctionData,
) {
  // Check for duplicate name in project
  const [existing] = await db
    .select()
    .from(functions)
    .where(and(eq(functions.projectId, projectId), eq(functions.name, data.name)))
    .limit(1);

  if (existing) {
    throw new Error(`Function with name "${data.name}" already exists in this project`);
  }

  const functionId = nanoid();
  const now = new Date();

  // Create function record
  const result = await db
    .insert(functions)
    .values({
      id: functionId,
      projectId,
      name: data.name,
      runtime: data.runtime || "bun",
      handler: data.handler,
      status: "draft",
      memory: data.memory,
      timeout: data.timeout,
      createdAt: now,
      updatedAt: now,
    })
    .returning();

  const func = Array.isArray(result) ? result[0] : null;
  if (!func) {
    throw new Error("Failed to create function");
  }

  // Create initial version if code is provided
  if (data.code) {
    const version = generateVersion();
    const { codePath, codeHash } = await storeFunctionCode(
      projectId,
      functionId,
      version,
      data.code,
    );

    await db.insert(functionVersions).values({
      id: nanoid(),
      functionId,
      version,
      codeHash,
      codePath,
      createdAt: now,
    });
  }

  return func;
}

/**
 * Update a function, creating a new version if code changed
 */
export async function updateFunction(
  functionId: string,
  projectId: string,
  data: UpdateFunctionData,
) {
  // Verify function exists and belongs to project
  const func = await getFunctionById(functionId, projectId);

  const updateData: Partial<typeof functions.$inferInsert> = {
    updatedAt: new Date(),
  };

  if (data.name !== undefined) {
    // Check for duplicate name if name is being changed
    if (data.name !== func.name) {
      const [existing] = await db
        .select()
        .from(functions)
        .where(
          and(
            eq(functions.projectId, projectId),
            eq(functions.name, data.name),
          ),
        )
        .limit(1);

      if (existing) {
        throw new Error(
          `Function with name "${data.name}" already exists in this project`,
        );
      }
    }
    updateData.name = data.name;
  }

  if (data.runtime !== undefined) {
    updateData.runtime = data.runtime;
  }

  if (data.handler !== undefined) {
    updateData.handler = data.handler;
  }

  if (data.memory !== undefined) {
    updateData.memory = data.memory;
  }

  if (data.timeout !== undefined) {
    updateData.timeout = data.timeout;
  }

  // Update function record
  const updateResult = await db
    .update(functions)
    .set(updateData)
    .where(eq(functions.id, functionId))
    .returning();

  const updated = Array.isArray(updateResult) ? updateResult[0] : null;
  if (!updated) {
    throw new Error("Failed to update function");
  }

  // Create new version if code is provided
  if (data.code !== undefined) {
    // Check if code actually changed by comparing with latest version
    const existingVersions = await getFunctionVersions(functionId, projectId);
    let codeChanged = true;

    if (existingVersions.length > 0) {
      const latestVersion = existingVersions[0];
      if (latestVersion) {
        const { readFunctionCode } = await import("./function-storage");
        try {
          const existingCode = await readFunctionCode(
            projectId,
            functionId,
            latestVersion.version,
          );
          codeChanged = existingCode !== data.code;
        } catch {
          // If we can't read existing code, assume it changed
          codeChanged = true;
        }
      }
    }

    if (codeChanged) {
      const version = generateVersion();
      const { codePath, codeHash } = await storeFunctionCode(
        projectId,
        functionId,
        version,
        data.code,
      );

      await db.insert(functionVersions).values({
        id: nanoid(),
        functionId,
        version,
        codeHash,
        codePath,
        createdAt: new Date(),
      });
    }
  }

  return updated;
}

/**
 * Delete a function (cascade will handle versions and deployments)
 */
export async function deleteFunction(functionId: string, projectId: string) {
  // Verify function exists and belongs to project
  await getFunctionById(functionId, projectId);

  // Delete function (cascade will handle related records)
  await db.delete(functions).where(eq(functions.id, functionId));

  // Also delete function code from filesystem
  const { deleteFunctionCode } = await import("./function-storage");
  await deleteFunctionCode(projectId, functionId);
}

/**
 * Get all versions for a function, ordered by creation date (newest first)
 */
export async function getFunctionVersions(
  functionId: string,
  projectId: string,
) {
  // Verify function exists and belongs to project
  await getFunctionById(functionId, projectId);

  return await db
    .select()
    .from(functionVersions)
    .where(eq(functionVersions.functionId, functionId))
    .orderBy(desc(functionVersions.createdAt));
}
