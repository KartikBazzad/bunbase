/**
 * Invoke function command
 */

import { Command } from "commander";
import { createServerClient } from "@bunbase/server-sdk";
import { loadAuth, getCookieHeader } from "../../utils/auth";
import { readFileSync, existsSync } from "fs";

export function createInvokeCommand(): Command {
  return new Command("invoke")
    .description("Invoke a function")
    .argument("<function-id>", "Function ID or name to invoke")
    .option("--data <data>", "JSON data to pass to function")
    .option("--data-file <file>", "File containing JSON data")
    .action(async (functionId, options) => {
      const auth = loadAuth();
      if (!auth?.apiKey && !auth?.user) {
        console.error("Error: Not authenticated. Run 'bunbase login' first.");
        process.exit(1);
      }

      const cookieHeader = auth.user ? getCookieHeader() : undefined;
      const client = createServerClient({
        apiKey: auth.apiKey,
        baseURL: auth.baseURL || "http://localhost:3000/api",
        projectId: auth.projectId,
        useCookies: !!auth.user,
        cookieHeader,
      });

      try {
        // Parse data
        let body: any = undefined;
        if (options.dataFile) {
          if (!existsSync(options.dataFile)) {
            console.error(`Error: File not found: ${options.dataFile}`);
            process.exit(1);
          }
          const content = readFileSync(options.dataFile, "utf-8");
          body = JSON.parse(content);
        } else if (options.data) {
          body = JSON.parse(options.data);
        }

        // If functionId is a name, find the function
        let actualFunctionId = functionId;
        const functions = await client.functions.list();
        const found = functions.find((f) => f.name === functionId || f.id === functionId);
        if (found) {
          actualFunctionId = found.id;
        }

        console.log(`Invoking function: ${actualFunctionId}...`);
        const result = await client.functions.invoke(actualFunctionId, { body });

        console.log("\nResult:");
        console.log(JSON.stringify(result.result, null, 2));
        console.log(`\nExecution time: ${result.executionTime}ms`);
      } catch (error: any) {
        console.error("Error invoking function:", error.message);
        process.exit(1);
      }
    });
}
