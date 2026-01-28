/**
 * Projects commands
 */

import { Command } from "commander";
import { loadAuth, apiRequest, getCookieHeader } from "../utils/auth";

export function createProjectsCommand(): Command {
  const projects = new Command("projects")
    .description("Manage projects");

  // List projects
  projects
    .command("list")
    .description("List all projects")
    .action(async () => {
      const auth = loadAuth();
      if (!auth?.user) {
        console.error("Error: Not authenticated. Run 'bunbase login' first.");
        process.exit(1);
      }

      const baseURL = auth.baseURL || "http://localhost:3001/api";
      try {
        const projects = await apiRequest("/projects", { method: "GET" }, baseURL);
        
        if (projects.length === 0) {
          console.log("No projects found.");
          console.log("Create a project with: bunbase projects create <name>");
          return;
        }

        console.log("\n📦 Your Projects:\n");
        projects.forEach((project: any) => {
          console.log(`  ${project.name}`);
          console.log(`    ID: ${project.id}`);
          console.log(`    Slug: ${project.slug}`);
          console.log(`    Created: ${new Date(project.created_at).toLocaleDateString()}`);
          console.log("");
        });
      } catch (error: any) {
        console.error(`❌ Failed to list projects: ${error.message}`);
        process.exit(1);
      }
    });

  // Create project
  projects
    .command("create")
    .description("Create a new project")
    .argument("<name>", "Project name")
    .action(async (name: string) => {
      const auth = loadAuth();
      if (!auth?.user) {
        console.error("Error: Not authenticated. Run 'bunbase login' first.");
        process.exit(1);
      }

      const baseURL = auth.baseURL || "http://localhost:3001/api";
      try {
        const project = await apiRequest(
          "/projects",
          {
            method: "POST",
            body: JSON.stringify({ name }),
          },
          baseURL
        );

        console.log("✅ Project created successfully!");
        console.log(`   Name: ${project.name}`);
        console.log(`   ID: ${project.id}`);
        console.log(`   Slug: ${project.slug}`);
        console.log(`\n   Set as active project:`);
        console.log(`   bunbase projects use ${project.id}`);
      } catch (error: any) {
        console.error(`❌ Failed to create project: ${error.message}`);
        process.exit(1);
      }
    });

  // Use project
  projects
    .command("use")
    .description("Set active project")
    .argument("<project-id>", "Project ID")
    .action(async (projectId: string) => {
      const auth = loadAuth();
      if (!auth?.user) {
        console.error("Error: Not authenticated. Run 'bunbase login' first.");
        process.exit(1);
      }

      const baseURL = auth.baseURL || "http://localhost:3001/api";
      try {
        // Verify project exists and user has access
        const project = await apiRequest(`/projects/${projectId}`, { method: "GET" }, baseURL);
        
        // Save active project ID
        const { saveAuth } = await import("../utils/auth");
        saveAuth({
          ...auth,
          projectId: projectId,
        });

        console.log("✅ Active project set!");
        console.log(`   Project: ${project.name} (${project.slug})`);
        console.log(`   You can now deploy functions to this project.`);
      } catch (error: any) {
        console.error(`❌ Failed to set active project: ${error.message}`);
        process.exit(1);
      }
    });

  return projects;
}
