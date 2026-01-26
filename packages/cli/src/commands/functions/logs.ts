/**
 * Function logs command
 */

import { Command } from "commander";
import { createServerClient } from "@bunbase/server-sdk";
import { loadAuth, getCookieHeader } from "../../utils/auth";

export function createLogsCommand(): Command {
  return new Command("logs")
    .description("View function logs")
    .argument("<function-id>", "Function ID or name")
    .option("--tail", "Stream logs in real-time")
    .option("--follow", "Follow logs (alias for --tail)")
    .option("--limit <number>", "Limit number of logs", "100")
    .option("--offset <number>", "Offset for pagination", "0")
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
        // If functionId is a name, find the function
        let actualFunctionId = functionId;
        const functions = await client.functions.list();
        const found = functions.find((f) => f.name === functionId || f.id === functionId);
        if (found) {
          actualFunctionId = found.id;
        }

        if (options.tail || options.follow) {
          console.log(`Streaming logs for function: ${actualFunctionId}...`);
          console.log("(Press Ctrl+C to stop)\n");

          // Poll for new logs
          let lastOffset = 0;
          const pollInterval = setInterval(async () => {
            try {
              const logs = await client.functions.getLogs(actualFunctionId, {
                limit: parseInt(options.limit),
                offset: lastOffset,
              });

              logs.logs.forEach((log) => {
                const timestamp = new Date(log.timestamp).toISOString();
                console.log(`[${timestamp}] [${log.level.toUpperCase()}] ${log.message}`);
                if (log.metadata) {
                  console.log(`  Metadata:`, JSON.stringify(log.metadata, null, 2));
                }
              });

              lastOffset += logs.logs.length;
            } catch (error: any) {
              console.error("Error fetching logs:", error.message);
            }
          }, 2000); // Poll every 2 seconds

          // Handle Ctrl+C
          process.on("SIGINT", () => {
            clearInterval(pollInterval);
            console.log("\nStopped streaming logs.");
            process.exit(0);
          });
        } else {
          const logs = await client.functions.getLogs(actualFunctionId, {
            limit: parseInt(options.limit),
            offset: parseInt(options.offset),
          });

          if (logs.logs.length === 0) {
            console.log("No logs found.");
            return;
          }

          console.log(`\nFound ${logs.total} log(s):\n`);
          logs.logs.forEach((log) => {
            const timestamp = new Date(log.timestamp).toISOString();
            console.log(`[${timestamp}] [${log.level.toUpperCase()}] ${log.message}`);
            if (log.metadata) {
              console.log(`  Metadata:`, JSON.stringify(log.metadata, null, 2));
            }
          });
        }
      } catch (error: any) {
        console.error("Error fetching logs:", error.message);
        process.exit(1);
      }
    });
}
