/**
 * Process-Based Function Executor
 * Executes functions in isolated processes (stronger isolation than workers)
 */

import { spawn } from "child_process";
import { join } from "path";
import type { WorkerResult } from "./function-worker";

export interface ProcessTask {
  functionId: string;
  version: string;
  codePath: string;
  request: {
    method: string;
    url: string;
    headers: Record<string, string>;
    body?: any;
  };
  env: Record<string, string>;
  timeout: number;
  memory?: number;
}

// ProcessResult uses the same interface as WorkerResult
export type ProcessResult = WorkerResult;

/**
 * Execute function in isolated process
 */
export async function executeInProcess(
  task: ProcessTask,
): Promise<ProcessResult> {
  const startTime = Date.now();
  const processScript = join(import.meta.dir, "function-process-script.ts");

  return new Promise((resolve) => {
    // Spawn Bun process with resource limits
    const child = spawn("bun", [processScript], {
      stdio: ["pipe", "pipe", "pipe"],
      env: {
        ...process.env,
        ...task.env,
        FUNCTION_CODE_PATH: task.codePath,
        FUNCTION_REQUEST: JSON.stringify(task.request),
        FUNCTION_TIMEOUT: String(task.timeout * 1000),
      },
    });

    let stdout = "";
    let stderr = "";
    const logs: Array<{ level: string; message: string; timestamp: Date }> = [];

    // Set timeout
    const timeout = setTimeout(() => {
      child.kill("SIGTERM");
      resolve({
        success: false,
        error: "Function execution timeout",
        logs,
        executionTime: Date.now() - startTime,
      });
    }, task.timeout * 1000);

    // Collect stdout
    child.stdout?.on("data", (data) => {
      stdout += data.toString();
    });

    // Collect stderr
    child.stderr?.on("data", (data) => {
      stderr += data.toString();
      logs.push({
        level: "error",
        message: data.toString(),
        timestamp: new Date(),
      });
    });

    // Handle process exit
    child.on("exit", (code, signal) => {
      clearTimeout(timeout);

      if (code !== 0 || signal) {
        resolve({
          success: false,
          error: stderr || `Process exited with code ${code}`,
          logs,
          executionTime: Date.now() - startTime,
        });
        return;
      }

      try {
        const result = JSON.parse(stdout);
        resolve({
          success: result.success,
          response: result.response,
          error: result.error,
          logs: [...logs, ...(result.logs || [])],
          executionTime: Date.now() - startTime,
        });
      } catch (error: any) {
        resolve({
          success: false,
          error: `Failed to parse process output: ${error.message}`,
          logs,
          executionTime: Date.now() - startTime,
        });
      }
    });

    // Send task data to process
    child.stdin?.write(JSON.stringify(task));
    child.stdin?.end();
  });
}
