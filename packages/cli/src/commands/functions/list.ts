/**
 * List functions command
 */

import { Command } from "commander";
import { createServerClient } from "@bunbase/server-sdk";
import { loadAuth, getCookieHeader } from "../../utils/auth";

export function createListCommand(): Command {
  return new Command("list")
    .description("List all functions in the project")
    .option("--json", "Output as JSON")
    .action(async (options) => {
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
        const functions = await client.functions.list();

        if (options.json) {
          console.log(JSON.stringify(functions, null, 2));
        } else {
          if (functions.length === 0) {
            console.log("No functions found.");
            return;
          }

          console.log(`\nFound ${functions.length} function(s):\n`);
          functions.forEach((fn) => {
            console.log(`  ${fn.id} - ${fn.name}`);
            console.log(`    Runtime: ${fn.runtime}`);
            console.log(`    Handler: ${fn.handler}`);
            if (fn.type) {
              console.log(`    Type: ${fn.type}`);
            }
            console.log("");
          });
        }
      } catch (error: any) {
        console.error("Error listing functions:", error.message);
        process.exit(1);
      }
    });
}
