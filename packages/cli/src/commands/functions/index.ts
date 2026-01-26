/**
 * Functions commands
 */

import { Command } from "commander";
import { createListCommand } from "./list";
import { createDeployCommand } from "./deploy";
import { createInvokeCommand } from "./invoke";
import { createLogsCommand } from "./logs";
import { createCreateCommand } from "./create";
import { createDeleteCommand } from "./delete";
import { createInitCommand } from "./init";

export function createFunctionsCommand(): Command {
  const functions = new Command("functions")
    .alias("fn")
    .description("Manage server functions");

  functions.addCommand(createInitCommand());
  functions.addCommand(createListCommand());
  functions.addCommand(createDeployCommand());
  functions.addCommand(createInvokeCommand());
  functions.addCommand(createLogsCommand());
  functions.addCommand(createCreateCommand());
  functions.addCommand(createDeleteCommand());

  return functions;
}
