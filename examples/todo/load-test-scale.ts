#!/usr/bin/env bun
/**
 * Sequential Scale Test Runner
 * 
 * Runs load tests with increasing user counts (100, 200, 300, ..., 1000)
 * to measure performance scaling.
 */

import { getScenario } from "./load-test-scenarios";
import { LoadTestRunner } from "./load-test-runner";
import { writeFile } from "fs/promises";

const scenarios = [
  "users100",
  "users200",
  "users300",
  "users400",
  "users500",
  "users600",
  "users700",
  "users800",
  "users900",
  "users1000",
];

interface TestResult {
  scenario: string;
  users: number;
  timestamp: string;
  results: {
    totalRequests: number;
    successfulRequests: number;
    failedRequests: number;
    successRate: number;
    totalDuration: number;
    requestsPerSecond: number;
    latency: {
      mean: number;
      median: number;
      p95: number;
      p99: number;
      min: number;
      max: number;
    };
    eventDelivery: {
      totalEvents: number;
      avgLatency: number;
    };
  };
}

async function main() {
  console.log("🚀 Starting Sequential Scale Test");
  console.log("===================================\n");
  console.log("This will run tests with increasing user counts:");
  console.log("100 → 200 → 300 → 400 → 500 → 600 → 700 → 800 → 900 → 1000\n");

  const allResults: TestResult[] = [];

  for (const scenarioName of scenarios) {
    const scenario = getScenario(scenarioName);
    if (!scenario) {
      console.error(`❌ Scenario not found: ${scenarioName}`);
      continue;
    }

    console.log(`\n${"=".repeat(60)}`);
    console.log(`Running: ${scenario.name}`);
    console.log(`Users: ${scenario.config.concurrentUsers}`);
    console.log(`${"=".repeat(60)}\n`);

      const runner = new LoadTestRunner(scenario.config);
    
    try {
      await runner.run();
      
      // Access metrics through the private field (using type assertion)
      // The metrics are already calculated in printResults()
      const metrics = (runner as any).metrics;
      const stats = metrics.getStatistics();
      const result: TestResult = {
        scenario: scenarioName,
        users: scenario.config.concurrentUsers,
        timestamp: new Date().toISOString(),
        results: stats,
      };

      allResults.push(result);

      // Print summary
      console.log(`\n📊 Summary for ${scenario.config.concurrentUsers} users:`);
      console.log(`   Success Rate: ${stats.successRate.toFixed(2)}%`);
      console.log(`   Throughput: ${stats.requestsPerSecond.toFixed(2)} req/s`);
      console.log(`   Mean Latency: ${stats.latency.mean.toFixed(2)}ms`);
      console.log(`   P95 Latency: ${stats.latency.p95.toFixed(2)}ms`);
      console.log(`   P99 Latency: ${stats.latency.p99.toFixed(2)}ms`);

      // Wait between tests to let the system recover
      if (scenarioName !== scenarios[scenarios.length - 1]) {
        console.log("\n⏳ Waiting 10 seconds before next test...\n");
        await new Promise((resolve) => setTimeout(resolve, 10000));
      }
    } catch (error) {
      console.error(`❌ Error running ${scenarioName}:`, error);
      continue;
    }
  }

  // Print final comparison
  console.log(`\n${"=".repeat(60)}`);
  console.log("📈 SCALE TEST RESULTS SUMMARY");
  console.log(`${"=".repeat(60)}\n`);

  console.log("Users | Success Rate | Throughput (req/s) | Mean Latency (ms) | P95 Latency (ms) | P99 Latency (ms)");
  console.log("-".repeat(100));

  for (const result of allResults) {
    const r = result.results;
    console.log(
      `${result.users.toString().padStart(5)} | ${r.successRate.toFixed(2).padStart(11)}% | ${r.requestsPerSecond.toFixed(2).padStart(17)} | ${r.latency.mean.toFixed(2).padStart(16)} | ${r.latency.p95.toFixed(2).padStart(15)} | ${r.latency.p99.toFixed(2).padStart(15)}`
    );
  }

  // Export results to JSON
  const filename = `scale-test-results-${Date.now()}.json`;
  await writeFile(filename, JSON.stringify(allResults, null, 2));
  console.log(`\n📁 Full results exported to: ${filename}`);

  // Calculate scaling metrics
  if (allResults.length >= 2) {
    const first = allResults[0].results;
    const last = allResults[allResults.length - 1].results;
    
    const throughputScaling = (last.requestsPerSecond / first.requestsPerSecond) * 100;
    const latencyIncrease = ((last.latency.mean - first.latency.mean) / first.latency.mean) * 100;
    
    console.log(`\n📊 Scaling Analysis:`);
    console.log(`   Throughput scaling: ${throughputScaling.toFixed(2)}% (${first.requestsPerSecond.toFixed(2)} → ${last.requestsPerSecond.toFixed(2)} req/s)`);
    console.log(`   Latency increase: ${latencyIncrease.toFixed(2)}% (${first.latency.mean.toFixed(2)}ms → ${last.latency.mean.toFixed(2)}ms)`);
  }
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
