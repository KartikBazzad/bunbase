/**
 * Create function command
 */

import { Command } from "commander";
import { createServerClient } from "@bunbase/server-sdk";
import { loadAuth, getCookieHeader } from "../../utils/auth";
import { readFileSync, existsSync } from "fs";
import { join } from "path";

export function createCreateCommand(): Command {
  return new Command("create")
    .description("Create a new function")
    .argument("<name>", "Function name")
    .requiredOption("--runtime <runtime>", "Runtime (nodejs20, bun, etc.)")
    .requiredOption("--handler <handler>", "Handler function (e.g., index.handler)")
    .option("--type <type>", "Function type (http or callable)", "http")
    .option("--path <path>", "HTTP path (for HTTP functions)", "/")
    .option("--methods <methods>", "HTTP methods (comma-separated)", "GET")
    .option("--code <file>", "Path to function code file")
    .option("--memory <mb>", "Memory in MB", "512")
    .option("--timeout <seconds>", "Timeout in seconds", "30")
    .action(async (name, options) => {
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
        // Read code if provided
        let code: string | undefined;
        if (options.code) {
          if (!existsSync(options.code)) {
            console.error(`Error: File not found: ${options.code}`);
            process.exit(1);
          }
          code = readFileSync(options.code, "utf-8");
        }

        console.log(`Creating function: ${name}...`);

        let result;
        if (options.type === "callable") {
          result = await client.functions.createCallableFunction({
            name,
            runtime: options.runtime as any,
            handler: options.handler,
            code,
            memory: parseInt(options.memory),
            timeout: parseInt(options.timeout),
          });
        } else {
          result = await client.functions.createHTTPFunction({
            name,
            runtime: options.runtime as any,
            handler: options.handler,
            path: options.path,
            methods: options.methods.split(",").map((m: string) => m.trim()),
            code,
            memory: parseInt(options.memory),
            timeout: parseInt(options.timeout),
          });
        }

        console.log(`✅ Function created successfully!`);
        console.log(`   ID: ${result.id}`);
        console.log(`   Name: ${result.name}`);
        console.log(`   Runtime: ${result.runtime}`);
      } catch (error: any) {
        console.error("Error creating function:", error.message);
        process.exit(1);
      }
    });
}
