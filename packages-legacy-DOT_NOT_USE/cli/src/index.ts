#!/usr/bin/env bun
/**
 * BunBase CLI
 */

import { Command } from "commander";
import { createFunctionsCommand } from "./commands/functions";
import { createProjectsCommand } from "./commands/projects";
import {
  loadAuth,
  saveAuth,
  clearAuth,
  loginWithEmail,
  getSession,
  saveSessionCookies,
  getCookieHeader,
} from "./utils/auth";
import { promptEmail, promptPassword } from "./utils/prompts";

const program = new Command();

program
  .name("bunbase")
  .description("BunBase CLI for managing projects and deploying functions")
  .version("0.1.0");

// Login command
program
  .command("login")
  .description("Login to BunBase Platform")
  .option("--api-key <key>", "API key (legacy)")
  .option("--email <email>", "Email address")
  .option("--password <password>", "Password")
  .option("--base-url <url>", "Base URL", "http://localhost:3001/api")
  .option("--project-id <id>", "Project ID")
  .action(async (options) => {
    const baseURL = options.baseUrl || "http://localhost:3001/api";

    if (options.apiKey) {
      // API key login (backward compatible)
      saveAuth({
        apiKey: options.apiKey,
        baseURL,
        projectId: options.projectId,
      });
      console.log("✅ Logged in successfully with API key!");
    } else {
      // Email/password login
      let email = options.email;
      let password = options.password;

      // Prompt for email if not provided
      if (!email) {
        email = await promptEmail();
      }

      // Prompt for password if not provided
      if (!password) {
        password = await promptPassword();
      }

      try {
        console.log("\n🔐 Logging in...");

        const result = await loginWithEmail(email, password, baseURL);

        // Save session cookies
        saveSessionCookies(result.cookies);

        // Save auth config
        saveAuth({
          baseURL,
          projectId: options.projectId,
          user: {
            id: result.user.id,
            email: result.user.email,
            name: result.user.name,
          },
        });

        console.log("✅ Logged in successfully!");
        console.log(`   User: ${result.user.name} (${result.user.email})`);
        if (options.projectId) {
          console.log(`   Project: ${options.projectId}`);
        }
      } catch (error: any) {
        console.error(`❌ Login failed: ${error.message}`);
        process.exit(1);
      }
    }
  });

// Logout command
program
  .command("logout")
  .description("Logout from BunBase Platform")
  .action(async () => {
    const auth = loadAuth();
    if (auth?.user) {
      // Try to call logout endpoint if we have session
      try {
        const baseURL = auth.baseURL || "http://localhost:3001/api";
        const url = new URL("/auth/logout", baseURL);
        const cookieHeader = getCookieHeader();

        if (cookieHeader) {
          await fetch(url.toString(), {
            method: "POST",
            headers: {
              Cookie: cookieHeader,
            },
            credentials: "include",
          });
        }
      } catch (error) {
        // Ignore errors during logout
      }
    }

    clearAuth();
    console.log("✅ Logged out successfully!");
  });

// Add projects commands
program.addCommand(createProjectsCommand());

// Add functions commands
program.addCommand(createFunctionsCommand());

// Parse arguments
program.parse();
