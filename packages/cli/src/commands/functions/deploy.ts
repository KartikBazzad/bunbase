/**
 * Deploy functions command
 */

import { Command } from "commander";
import { createServerClient } from "@bunbase/server-sdk";
import { loadAuth, getCookieHeader } from "../../utils/auth";
import { loadConfig } from "../../utils/config";
import { readFileSync, existsSync, readdirSync, statSync } from "fs";
import { join } from "path";

export function createDeployCommand(): Command {
  return new Command("deploy")
    .description("Deploy a function or all functions")
    .argument("[function-name]", "Name of the function to deploy")
    .option("--dir <directory>", "Deploy all functions from directory")
    .option("--source <path>", "Source directory for function code")
    .action(async (functionName, options) => {
      const auth = loadAuth();
      if (!auth?.apiKey && !auth?.user) {
        console.error("Error: Not authenticated. Run 'bunbase login' first.");
        process.exit(1);
      }

      // Use session-based auth if user is logged in, otherwise use API key
      const cookieHeader = auth.user ? getCookieHeader() : undefined;
      const client = createServerClient({
        apiKey: auth.apiKey,
        baseURL: auth.baseURL || "http://localhost:3000/api",
        projectId: auth.projectId,
        useCookies: !!auth.user, // Use cookies if user is logged in
        cookieHeader, // Pass cookie header for session auth
      });

      try {
        const config = loadConfig();

        if (options.dir) {
          // Deploy all functions from directory
          console.log(`Deploying functions from ${options.dir}...`);
          const dirPath = join(process.cwd(), options.dir);
          if (!existsSync(dirPath)) {
            console.error(`Error: Directory not found: ${dirPath}`);
            process.exit(1);
          }

          const functions = readdirSync(dirPath)
            .filter((item) => {
              const itemPath = join(dirPath, item);
              return statSync(itemPath).isDirectory();
            });

          if (functions.length === 0) {
            console.log("No functions found in directory.");
            return;
          }

          console.log(`Found ${functions.length} function(s) to deploy:\n`);

          for (const fnName of functions) {
            try {
              await deployFunction(fnName, config, client, options);
            } catch (error: any) {
              console.error(`  ❌ Failed to deploy ${fnName}: ${error.message}\n`);
            }
          }
        } else if (functionName) {
          // Deploy specific function
          console.log(`Deploying function: ${functionName}...`);

          // Find function in config
          if (config?.functions?.[functionName]) {
            const fnConfig = config.functions[functionName];
            const sourceDir = options.source || join("functions", functionName);

            // Read function code
            let code: string | undefined;
            const handlerPath = join(sourceDir, fnConfig.handler.split(".")[0] + ".ts");
            if (existsSync(handlerPath)) {
              code = readFileSync(handlerPath, "utf-8");
            }

            // Check if function exists
            const functions = await client.functions.list();
            const existing = functions.find((f) => f.name === functionName);

            let functionId: string;
            if (existing) {
              // Update existing function
              console.log("Updating existing function...");
              await client.functions.update(existing.id, {
                code,
                memory: fnConfig.memory,
                timeout: fnConfig.timeout,
              });
              functionId = existing.id;
            } else {
              // Create new function
              console.log("Creating new function...");
              if (fnConfig.type === "http") {
                const result = await client.functions.createHTTPFunction({
                  name: functionName,
                  runtime: fnConfig.runtime as any,
                  handler: fnConfig.handler,
                  path: fnConfig.path || `/${functionName}`,
                  methods: fnConfig.methods || ["GET"],
                  code,
                  memory: fnConfig.memory,
                  timeout: fnConfig.timeout,
                });
                functionId = result.id;
              } else {
                const result = await client.functions.createCallableFunction({
                  name: functionName,
                  runtime: fnConfig.runtime as any,
                  handler: fnConfig.handler,
                  code,
                  memory: fnConfig.memory,
                  timeout: fnConfig.timeout,
                });
                functionId = result.id;
              }
            }

            // Deploy function
            console.log("Deploying...");
            const deployResult = await client.functions.deploy(functionId);
            console.log(`✅ Function deployed successfully!`);
            console.log(`   Version: ${deployResult.version}`);
          } else {
            console.error(`Error: Function '${functionName}' not found in config.`);
            process.exit(1);
          }
        } else {
          // Deploy all functions from config
          if (!config?.functions) {
            console.error("Error: No functions found in config.");
            process.exit(1);
          }

          console.log(`Deploying ${Object.keys(config.functions).length} function(s)...\n`);

          for (const [name] of Object.entries(config.functions)) {
            try {
              await deployFunction(name, config, client, options);
            } catch (error: any) {
              console.error(`  ❌ Failed to deploy ${name}: ${error.message}\n`);
            }
          }
        }
      } catch (error: any) {
        console.error("Error deploying function:", error.message);
        process.exit(1);
      }
    });
}

/**
 * Deploy a single function
 */
async function deployFunction(
  functionName: string,
  config: any,
  client: any,
  options: any,
): Promise<void> {
  console.log(`Deploying function: ${functionName}...`);

  // Find function in config or auto-detect from functions directory
  let fnConfig = config?.functions?.[functionName];
  const sourceDir = options.source || join("functions", functionName);

  // Auto-detect if not in config
  if (!fnConfig && existsSync(sourceDir)) {
    // Try to read from functions directory
    const possibleFiles = ["index.ts", "index.js", "index.py", "handler.ts", "handler.js"];
    let handlerFile: string | undefined;
    let runtime: string = "nodejs20";

    for (const file of possibleFiles) {
      const filePath = join(sourceDir, file);
      if (existsSync(filePath)) {
        handlerFile = file;
        if (file.endsWith(".py")) {
          runtime = "python3.11";
        }
        break;
      }
    }

    if (handlerFile) {
      fnConfig = {
        runtime,
        handler: handlerFile.replace(/\.(ts|js|py)$/, ".handler"),
        type: "http",
        path: `/${functionName}`,
        methods: ["GET", "POST"],
      };
    }
  }

  if (!fnConfig) {
    throw new Error(`Function '${functionName}' not found in config or functions directory.`);
  }

  // Read function code
  let code: string | undefined;
  const handlerName = fnConfig.handler.split(".")[0];
  const extensions = [".ts", ".js", ".py"];
  for (const ext of extensions) {
    const handlerPath = join(sourceDir, handlerName + ext);
    if (existsSync(handlerPath)) {
      code = readFileSync(handlerPath, "utf-8");
      break;
    }
  }

  if (!code) {
    throw new Error(`Handler file not found for function '${functionName}'.`);
  }

  // Check if function exists
  const functions = await client.functions.list();
  const existing = functions.find((f) => f.name === functionName);

  let functionId: string;
  if (existing) {
    // Update existing function
    console.log("  Updating existing function...");
    await client.functions.update(existing.id, {
      code,
      memory: fnConfig.memory,
      timeout: fnConfig.timeout,
    });
    functionId = existing.id;
  } else {
    // Create new function
    console.log("  Creating new function...");
    if (fnConfig.type === "http") {
      const result = await client.functions.createHTTPFunction({
        name: functionName,
        runtime: fnConfig.runtime as any,
        handler: fnConfig.handler,
        path: fnConfig.path || `/${functionName}`,
        methods: fnConfig.methods || ["GET"],
        code,
        memory: fnConfig.memory,
        timeout: fnConfig.timeout,
      });
      functionId = result.id;
    } else {
      const result = await client.functions.createCallableFunction({
        name: functionName,
        runtime: fnConfig.runtime as any,
        handler: fnConfig.handler,
        code,
        memory: fnConfig.memory,
        timeout: fnConfig.timeout,
      });
      functionId = result.id;
    }
  }

  // Deploy function
  console.log("  Deploying...");
  const deployResult = await client.functions.deploy(functionId);
  console.log(`  ✅ Function deployed successfully!`);
  console.log(`     Version: ${deployResult.version}\n`);
}
