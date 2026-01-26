/**
 * Function Executor
 * Orchestrates function execution using worker threads
 */

import { getWorkerPool, WorkerTask, WorkerResult } from "./function-worker";
import { executeInProcess, ProcessTask } from "./function-process-executor";
import { getFunctionById } from "./function-helpers";
import { readFunctionCode } from "./function-storage";
import { db, functionEnvironments, functionLogs, functionMetrics } from "../db";
import { functionMetricsMinute } from "../db/schema";
import { eq, and } from "drizzle-orm";
import { decrypt } from "./encryption";
import { nanoid } from "nanoid";
import { logProjectOperation } from "./project-logger-utils";
import { getLogBuffer, LogEntry } from "./function-log-buffer";
import { getConcurrencyController } from "./function-concurrency";

export interface ExecutionRequest {
  method: string;
  url: string;
  headers: Record<string, string>;
  body?: any;
}

export interface ExecutionResult {
  success: boolean;
  response?: {
    status: number;
    headers: Record<string, string>;
    body: any;
  };
  error?: string;
  executionId: string;
  executionTime: number;
}

/**
 * Execute a function
 */
export async function executeFunction(
  functionId: string,
  projectId: string,
  request: ExecutionRequest,
): Promise<ExecutionResult> {
  const executionId = nanoid();
  const startTime = Date.now();

  try {
    // Get function details
    const func = await getFunctionById(functionId, projectId);

    if (func.status !== "deployed") {
      throw new Error("Function is not deployed");
    }

    // Check concurrency limits
    const concurrencyController = getConcurrencyController();
    const maxConcurrency = func.maxConcurrentExecutions || 10;
    const acquired = await concurrencyController.acquire(
      functionId,
      maxConcurrency,
    );

    if (!acquired) {
      throw new Error(
        "Function concurrency limit exceeded. Please try again later.",
      );
    }

    // Ensure we release concurrency permit on exit
    let released = false;
    const releaseConcurrency = () => {
      if (!released) {
        concurrencyController.release(functionId);
        released = true;
      }
    };

    // Get active version from function record
    if (!func.activeVersionId) {
      throw new Error("No active version found for function");
    }

    // Get version details
    const { functionVersions } = await import("../db/schema");
    const [version] = await db
      .select()
      .from(functionVersions)
      .where(eq(functionVersions.id, func.activeVersionId))
      .limit(1);

    if (!version) {
      throw new Error("Active version not found");
    }

    const codePath = version.codePath;

    // Get environment variables
    const envVars = await db
      .select()
      .from(functionEnvironments)
      .where(eq(functionEnvironments.functionId, functionId));

    // Decrypt environment variables
    const decryptedEnv: Record<string, string> = {};
    for (const env of envVars) {
      decryptedEnv[env.key] = env.isSecret
        ? await decrypt(env.value)
        : env.value;
    }

    // Choose execution method based on runtimeType
    const runtimeType = func.runtimeType || "worker";
    let result: WorkerResult;

    if (runtimeType === "process") {
      // Execute in isolated process
      const processTask: ProcessTask = {
        functionId,
        version: version.version,
        codePath,
        request,
        env: decryptedEnv,
        timeout: func.timeout || 30,
        memory: func.memory,
      };
      result = await executeInProcess(processTask);
    } else {
      // Execute in worker (default)
      const task: WorkerTask = {
        functionId,
        version: version.version,
        codePath,
        request,
        env: decryptedEnv,
        timeout: func.timeout || 30,
        memory: func.memory,
      };
      const workerPool = getWorkerPool();
      result = await workerPool.execute(task);
    }

    // Log execution (async buffered)
    logExecution(functionId, executionId, result, startTime);

    // Log to project logger
    logProjectOperation(projectId, "function_execution", {
      functionId,
      executionId,
      success: result.success,
      executionTime: result.executionTime,
    });

    // Update metrics
    await updateMetrics(functionId, result, startTime);

    // Release concurrency permit
    releaseConcurrency();

    return {
      success: result.success,
      response: result.response,
      error: result.error,
      executionId,
      executionTime: result.executionTime,
    };
  } catch (error: any) {
    // Release concurrency permit on error
    const concurrencyController = getConcurrencyController();
    concurrencyController.release(functionId);

    // Log error (async buffered)
    const logBuffer = getLogBuffer();
    logBuffer.append({
      id: nanoid(),
      functionId,
      executionId,
      level: "error",
      message: error.message || "Execution failed",
      metadata: { error: error.toString() },
      timestamp: new Date(),
    });

    return {
      success: false,
      error: error.message || "Execution failed",
      executionId,
      executionTime: Date.now() - startTime,
    };
  }
}

/**
 * Log function execution (async buffered)
 */
function logExecution(
  functionId: string,
  executionId: string,
  result: WorkerResult,
  startTime: number,
): void {
  const logBuffer = getLogBuffer();

  // Log execution start
  logBuffer.append({
    id: nanoid(),
    functionId,
    executionId,
    level: "info",
    message: "Function execution started",
    metadata: { startTime: new Date(startTime) },
    timestamp: new Date(startTime),
  });

  // Log function logs
  for (const log of result.logs) {
    logBuffer.append({
      id: nanoid(),
      functionId,
      executionId,
      level: log.level,
      message: log.message,
      metadata: { timestamp: log.timestamp },
      timestamp: log.timestamp,
    });
  }

  // Log execution result
  logBuffer.append({
    id: nanoid(),
    functionId,
    executionId,
    level: result.success ? "info" : "error",
    message: result.success
      ? "Function execution completed"
      : `Function execution failed: ${result.error}`,
    metadata: {
      success: result.success,
      executionTime: result.executionTime,
      responseStatus: result.response?.status,
    },
    timestamp: new Date(),
  });
}

/**
 * Update function metrics (minute-level and daily)
 */
async function updateMetrics(
  functionId: string,
  result: WorkerResult,
  startTime: number,
): Promise<void> {
  const now = new Date();
  const isColdStart = result.executionTime > 500;

  // Round timestamp to minute
  const minuteTimestamp = new Date(now);
  minuteTimestamp.setSeconds(0, 0);

  // Update minute-level metrics
  const [existingMinute] = await db
    .select()
    .from(functionMetricsMinute)
    .where(
      and(
        eq(functionMetricsMinute.functionId, functionId),
        eq(functionMetricsMinute.timestamp, minuteTimestamp),
      ),
    )
    .limit(1);

  if (existingMinute) {
    await db
      .update(functionMetricsMinute)
      .set({
        invocations: existingMinute.invocations + 1,
        errors: existingMinute.errors + (result.success ? 0 : 1),
        totalDuration: existingMinute.totalDuration + result.executionTime,
        coldStarts: existingMinute.coldStarts + (isColdStart ? 1 : 0),
      })
      .where(eq(functionMetricsMinute.id, existingMinute.id));
  } else {
    await db.insert(functionMetricsMinute).values({
      id: nanoid(),
      functionId,
      timestamp: minuteTimestamp,
      invocations: 1,
      errors: result.success ? 0 : 1,
      totalDuration: result.executionTime,
      coldStarts: isColdStart ? 1 : 0,
    });
  }

  // Update daily metrics (for backward compatibility and rollup)
  const today = new Date(now);
  today.setHours(0, 0, 0, 0);

  const [existing] = await db
    .select()
    .from(functionMetrics)
    .where(
      and(
        eq(functionMetrics.functionId, functionId),
        eq(functionMetrics.date, today),
      ),
    )
    .limit(1);

  if (existing) {
    await db
      .update(functionMetrics)
      .set({
        invocations: existing.invocations + 1,
        errors: existing.errors + (result.success ? 0 : 1),
        totalDuration: existing.totalDuration + result.executionTime,
        coldStarts: existing.coldStarts + (isColdStart ? 1 : 0),
      })
      .where(eq(functionMetrics.id, existing.id));
  } else {
    await db.insert(functionMetrics).values({
      id: nanoid(),
      functionId,
      date: today,
      invocations: 1,
      errors: result.success ? 0 : 1,
      totalDuration: result.executionTime,
      coldStarts: isColdStart ? 1 : 0,
    });
  }
}
