/**
 * Process-Based Function Executor
 * Executes functions in isolated processes (stronger isolation than workers)
 */

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

  return new Promise(async (resolve) => {
    // Spawn Bun process with resource limits using Bun.spawn
    const proc = Bun.spawn(["bun", processScript], {
      stdin: "pipe",
      stdout: "pipe",
      stderr: "pipe",
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
      proc.kill();
      resolve({
        success: false,
        error: "Function execution timeout",
        logs,
        executionTime: Date.now() - startTime,
      });
    }, task.timeout * 1000);

    // Collect stdout
    const stdoutReader = proc.stdout.getReader();
    (async () => {
      try {
        while (true) {
          const { done, value } = await stdoutReader.read();
          if (done) break;
          stdout += new TextDecoder().decode(value);
        }
      } catch (error) {
        // Stream closed
      }
    })();

    // Collect stderr
    const stderrReader = proc.stderr.getReader();
    (async () => {
      try {
        while (true) {
          const { done, value } = await stderrReader.read();
          if (done) break;
          const text = new TextDecoder().decode(value);
          stderr += text;
          logs.push({
            level: "error",
            message: text,
            timestamp: new Date(),
          });
        }
      } catch (error) {
        // Stream closed
      }
    })();

    // Wait for process to exit
    const exitCode = await proc.exited;
    clearTimeout(timeout);

    if (exitCode !== 0) {
      resolve({
        success: false,
        error: stderr || `Process exited with code ${exitCode}`,
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
}
