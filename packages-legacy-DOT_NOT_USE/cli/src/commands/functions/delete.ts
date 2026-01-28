/**
 * Delete function command
 */

import { Command } from "commander";
import { createServerClient } from "@bunbase/server-sdk";
import { loadAuth, getCookieHeader } from "../../utils/auth";

export function createDeleteCommand(): Command {
  return new Command("delete")
    .description("Delete a function")
    .argument("<function-id>", "Function ID or name to delete")
    .option("--force", "Skip confirmation")
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
        } else {
          console.error(`Error: Function '${functionId}' not found.`);
          process.exit(1);
        }

        if (!options.force) {
          // In a real implementation, you'd use a prompt library
          console.log(`Are you sure you want to delete function '${functionId}'? (y/N)`);
          // For now, we'll require --force flag
          console.error("Use --force flag to confirm deletion.");
          process.exit(1);
        }

        console.log(`Deleting function: ${actualFunctionId}...`);
        await client.functions.delete(actualFunctionId);
        console.log(`✅ Function deleted successfully!`);
      } catch (error: any) {
        console.error("Error deleting function:", error.message);
        process.exit(1);
      }
    });
}
