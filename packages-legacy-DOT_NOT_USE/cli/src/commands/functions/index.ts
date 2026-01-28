/**
 * Functions commands
 */

import { Command } from "commander";
import { createDeployCommand } from "./deploy";

export function createFunctionsCommand(): Command {
  const functions = new Command("functions")
    .description("Manage functions");

  functions.addCommand(createDeployCommand());

  return functions;
}
