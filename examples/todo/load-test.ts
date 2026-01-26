#!/usr/bin/env bun
/**
 * Load Test Runner CLI
 * 
 * Run load tests with various scenarios and profiles.
 */

import { getConfig, listProfiles } from "./load-test-config";
import { getScenario, listScenarios } from "./load-test-scenarios";
import { LoadTestRunner } from "./load-test-runner";

const args = process.argv.slice(2);

async function main() {
  const scenarioArg = args.find((arg) => arg.startsWith("--scenario="));
  const profileArg = args.find((arg) => arg.startsWith("--profile="));
  const listArg = args.includes("--list");

  if (listArg) {
    console.log("\nAvailable Scenarios:");
    console.log("====================");
    for (const name of listScenarios()) {
      const scenario = getScenario(name);
      if (scenario) {
        console.log(`  ${name}: ${scenario.description}`);
      }
    }

    console.log("\nAvailable Profiles:");
    console.log("==================");
    for (const name of listProfiles()) {
      const profile = getConfig(name);
      if (profile) {
        console.log(`  ${name}: ${profile.description}`);
      }
    }
    return;
  }

  let config;

  if (scenarioArg) {
    const scenarioName = scenarioArg.split("=")[1];
    const scenario = getScenario(scenarioName);
    if (!scenario) {
      console.error(`❌ Unknown scenario: ${scenarioName}`);
      console.log(`Available scenarios: ${listScenarios().join(", ")}`);
      process.exit(1);
    }
    config = scenario.config;
  } else if (profileArg) {
    const profileName = profileArg.split("=")[1];
    const profile = getConfig(profileName);
    if (!profile) {
      console.error(`❌ Unknown profile: ${profileName}`);
      console.log(`Available profiles: ${listProfiles().join(", ")}`);
      process.exit(1);
    }
    config = profile;
  } else {
    // Default to CRUD scenario
    const scenario = getScenario("crud");
    if (!scenario) {
      console.error("❌ Default scenario not found");
      process.exit(1);
    }
    config = scenario.config;
  }

  const runner = new LoadTestRunner(config);
  await runner.run();

  // Export results if requested
  const exportArg = args.find((arg) => arg.startsWith("--export="));
  if (exportArg) {
    const format = exportArg.split("=")[1] as "json" | "csv";
    const results = runner.exportResults(format);
    const filename = `load-test-results-${Date.now()}.${format}`;
    await Bun.write(filename, results);
    console.log(`\n📁 Results exported to: ${filename}`);
  }
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
