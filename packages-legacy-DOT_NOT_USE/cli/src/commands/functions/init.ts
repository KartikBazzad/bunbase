/**
 * Initialize function command
 */

import { Command } from "commander";
import { bootstrapFunction } from "../../utils/templates";
import { loadConfig, getProjectConfig } from "../../utils/config";
import { writeFileSync, existsSync } from "fs";
import { join } from "path";
import type { FunctionRuntime, FunctionType } from "@bunbase/server-sdk";

export function createInitCommand(): Command {
  return new Command("init")
    .description("Initialize a new function with template code")
    .argument("<function-name>", "Name of the function to create")
    .requiredOption("--runtime <runtime>", "Runtime (nodejs20, bun, python3.11, etc.)")
    .option("--type <type>", "Function type (http or callable)", "http")
    .option("--handler <handler>", "Handler function name", "handler")
    .option("--path <path>", "HTTP path (for HTTP functions)")
    .option("--methods <methods>", "HTTP methods (comma-separated)", "GET,POST")
    .action(async (functionName, options) => {
      const projectRoot = process.cwd();

      // Validate function name
      if (!/^[a-z0-9-_]+$/i.test(functionName)) {
        console.error(
          "Error: Function name can only contain letters, numbers, hyphens, and underscores",
        );
        process.exit(1);
      }

      // Validate runtime
      const validRuntimes: FunctionRuntime[] = [
        "nodejs18",
        "nodejs20",
        "nodejs22",
        "bun",
        "python3.10",
        "python3.11",
        "python3.12",
        "go",
        "deno",
      ];

      if (!validRuntimes.includes(options.runtime as FunctionRuntime)) {
        console.error(
          `Error: Invalid runtime. Valid runtimes: ${validRuntimes.join(", ")}`,
        );
        process.exit(1);
      }

      // Validate type
      if (options.type !== "http" && options.type !== "callable") {
        console.error('Error: Type must be "http" or "callable"');
        process.exit(1);
      }

      try {
        console.log(`\n🚀 Initializing function: ${functionName}\n`);

        // Bootstrap function
        bootstrapFunction(projectRoot, {
          name: functionName,
          runtime: options.runtime as FunctionRuntime,
          type: options.type as FunctionType,
          handler: options.handler,
          path: options.path || `/${functionName}`,
          methods: options.methods
            ? options.methods.split(",").map((m: string) => m.trim())
            : ["GET", "POST"],
        });

        // Update bunbase.config.ts or functions.json
        const config = loadConfig(projectRoot);
        const projectConfig = getProjectConfig(projectRoot);

        const functionConfig: any = {
          runtime: options.runtime,
          handler: options.handler,
          type: options.type,
        };

        if (options.type === "http") {
          functionConfig.path = options.path || `/${functionName}`;
          functionConfig.methods = options.methods
            ? options.methods.split(",").map((m: string) => m.trim())
            : ["GET", "POST"];
        }

        // Try to update bunbase.config.ts
        const configPath = join(projectRoot, "bunbase.config.ts");
        if (existsSync(configPath)) {
          // For TypeScript config, we'd need a proper parser
          // For now, we'll update functions.json if it exists
          const jsonConfigPath = join(projectRoot, "functions.json");
          if (existsSync(jsonConfigPath)) {
            const jsonConfig = JSON.parse(
              require("fs").readFileSync(jsonConfigPath, "utf-8"),
            );
            if (!jsonConfig.functions) {
              jsonConfig.functions = {};
            }
            jsonConfig.functions[functionName] = functionConfig;
            writeFileSync(
              jsonConfigPath,
              JSON.stringify(jsonConfig, null, 2),
            );
          } else {
            // Create functions.json
            writeFileSync(
              jsonConfigPath,
              JSON.stringify({ functions: { [functionName]: functionConfig } }, null, 2),
            );
          }
        } else {
          // Create functions.json
          const jsonConfigPath = join(projectRoot, "functions.json");
          writeFileSync(
            jsonConfigPath,
            JSON.stringify({ functions: { [functionName]: functionConfig } }, null, 2),
          );
        }

        console.log(`✅ Function initialized successfully!\n`);
        console.log(`📁 Function directory: functions/${functionName}/`);
        console.log(`📝 Edit the function code and then deploy with:`);
        console.log(`   bunbase functions deploy ${functionName}\n`);
      } catch (error: any) {
        console.error(`Error initializing function: ${error.message}`);
        process.exit(1);
      }
    });
}
