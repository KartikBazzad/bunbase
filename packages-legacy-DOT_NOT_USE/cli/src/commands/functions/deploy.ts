/**
 * Deploy functions command
 */

import { Command } from "commander";
import { loadAuth, apiRequest, getCookieHeader } from "../../utils/auth";
import { readFileSync, existsSync } from "fs";
import { join } from "path";

export function createDeployCommand(): Command {
  return new Command("deploy")
    .description("Deploy a function to the active project")
    .argument("[function-name]", "Name of the function to deploy")
    .option("--file <path>", "Path to function file")
    .option("--runtime <runtime>", "Runtime (bun or quickjs-ng)", "bun")
    .option("--handler <handler>", "Handler name", "default")
    .option("--version <version>", "Version tag", "v1")
    .action(async (functionName, options) => {
      const auth = loadAuth();
      if (!auth?.user) {
        console.error("Error: Not authenticated. Run 'bunbase login' first.");
        process.exit(1);
      }

      if (!auth.projectId) {
        console.error("Error: No active project. Run 'bunbase projects use <project-id>' first.");
        process.exit(1);
      }

      const baseURL = auth.baseURL || "http://localhost:3001/api";

      try {
        // Determine function name and file path
        let fnName = functionName;
        let filePath = options.file;

        if (!fnName && !filePath) {
          console.error("Error: Function name or --file option required");
          process.exit(1);
        }

        if (!filePath && fnName) {
          // Try to find function file
          const possiblePaths = [
            join(process.cwd(), `${fnName}.ts`),
            join(process.cwd(), `${fnName}.js`),
            join(process.cwd(), "functions", `${fnName}.ts`),
            join(process.cwd(), "functions", `${fnName}.js`),
          ];

          for (const path of possiblePaths) {
            if (existsSync(path)) {
              filePath = path;
              break;
            }
          }

          if (!filePath) {
            console.error(`Error: Function file not found for '${fnName}'`);
            console.error("   Tried:", possiblePaths.join(", "));
            console.error("   Use --file <path> to specify the file location");
            process.exit(1);
          }
        }

        if (!fnName && filePath) {
          // Extract function name from file path
          const basename = filePath.split("/").pop()?.split("\\").pop() || "";
          fnName = basename.replace(/\.(ts|js)$/, "");
        }

        if (!existsSync(filePath)) {
          console.error(`Error: File not found: ${filePath}`);
          process.exit(1);
        }

        console.log(`\n📦 Deploying function '${fnName}' to project ${auth.projectId}...`);
        console.log(`   File: ${filePath}`);
        console.log(`   Runtime: ${options.runtime}`);
        console.log(`   Handler: ${options.handler}`);
        console.log(`   Version: ${options.version}`);

        // Read function file
        const functionCode = readFileSync(filePath, "utf-8");

        // Bundle function (simplified - in production, use esbuild/bun)
        // For now, we'll send the code as-is and let the backend handle bundling
        // In a real implementation, you'd bundle it here
        const bundle = Buffer.from(functionCode).toString("base64");

        // Deploy via platform API
        const functionData = await apiRequest(
          `/projects/${auth.projectId}/functions`,
          {
            method: "POST",
            body: JSON.stringify({
              name: fnName,
              runtime: options.runtime,
              handler: options.handler,
              version: options.version,
              bundle: bundle,
            }),
          },
          baseURL
        );

        console.log("\n✅ Function deployed successfully!");
        console.log(`   Function ID: ${functionData.id}`);
        console.log(`   Service ID: ${functionData.function_service_id}`);
        console.log(`   Runtime: ${functionData.runtime}`);
        console.log(`\n   Invoke at: http://localhost:8080/functions/${functionData.function_service_id}`);
      } catch (error: any) {
        console.error(`\n❌ Deployment failed: ${error.message}`);
        process.exit(1);
      }
    });
}
