/**
 * Load Test Runner
 * 
 * Orchestrates parallel test execution, results aggregation, and reporting.
 */

import { createClient } from "@bunbase/js-sdk";
import type { LoadTestConfig } from "./load-test-config";
import { MetricsCollector } from "./performance-metrics";

const API_KEY = process.env.BUNBASE_API_KEY || "bunbase_pk_live_cxaZTm1TIJ9eQZYOUm2QVF48YuiTkufr";
const BASE_URL = process.env.BUNBASE_URL || "http://localhost:3000/api";

interface TestClient {
  id: number;
  client: ReturnType<typeof createClient>;
  documentIds: string[];
  eventUnsubscribe?: () => void;
  eventCount: number;
  lastEventTime: number;
}

export class LoadTestRunner {
  private config: LoadTestConfig;
  private metrics: MetricsCollector;
  private clients: TestClient[] = [];

  constructor(config: LoadTestConfig) {
    this.config = config;
    this.metrics = new MetricsCollector();
  }

  async run(): Promise<void> {
    console.log(`\n🚀 Starting load test: ${this.config.name}`);
    console.log(`   Description: ${this.config.description}`);
    console.log(`   Concurrent users: ${this.config.concurrentUsers}`);
    console.log(`   Duration: ${this.config.duration}ms`);
    console.log(`   Collection: ${this.config.collectionName}\n`);

    // Initialize clients
    for (let i = 0; i < this.config.concurrentUsers; i++) {
      const client = createClient({
        apiKey: API_KEY,
        baseURL: BASE_URL,
      });
      const testClient: TestClient = {
        id: i,
        client,
        documentIds: [],
        eventCount: 0,
        lastEventTime: 0,
        eventUnsubscribe: undefined,
      };
      
      // Subscribe to real-time events for a subset of clients (to avoid overwhelming WebSocket connections)
      // Only subscribe 10% of clients to real-time events
      if (i % 10 === 0) {
        const bunstore = client.bunstore();
        const collection = bunstore.collection(this.config.collectionName);
        const operationStartTime = Date.now();
        
        testClient.eventUnsubscribe = collection.onSnapshot((snapshot: any) => {
          const eventTime = Date.now();
          const deliveryLatency = eventTime - operationStartTime;
          
          const changes = snapshot.docChanges ? snapshot.docChanges() : [];
          changes.forEach((change: any) => {
            let eventType: "INSERT" | "UPDATE" | "DELETE";
            if (change.type === "added") {
              eventType = "INSERT";
            } else if (change.type === "modified") {
              eventType = "UPDATE";
            } else {
              eventType = "DELETE";
            }
            
            this.metrics.recordEvent(eventType, deliveryLatency);
            testClient.eventCount++;
            testClient.lastEventTime = eventTime;
          });
        });
      }
      
      this.clients.push(testClient);
    }

    // Calculate operation distribution
    const totalOps = Object.values(this.config.operationMix).reduce((a, b) => a + b, 0);
    const opWeights = {
      create: this.config.operationMix.create / totalOps,
      read: this.config.operationMix.read / totalOps,
      update: this.config.operationMix.update / totalOps,
      delete: this.config.operationMix.delete / totalOps,
    };

    // Start all clients
    const startTime = Date.now();
    const endTime = startTime + this.config.duration;

    const promises = this.clients.map((testClient) =>
      this.runClient(testClient, opWeights, endTime)
    );

    // Wait for all clients to finish
    await Promise.all(promises);

    // Clean up event subscriptions
    for (const testClient of this.clients) {
      if (testClient.eventUnsubscribe) {
        testClient.eventUnsubscribe();
      }
    }

    // Wait a bit for any pending events to arrive
    await new Promise((resolve) => setTimeout(resolve, 1000));

    this.metrics.finish();

    // Print results
    this.printResults();
  }

  private async runClient(
    testClient: TestClient,
    opWeights: Record<string, number>,
    endTime: number,
  ): Promise<void> {
    const bunstore = testClient.client.bunstore();
    const collection = bunstore.collection(this.config.collectionName);

    while (Date.now() < endTime) {
      const rand = Math.random();
      let operation: "create" | "read" | "update" | "delete";

      const createWeight = opWeights.create || 0;
      const readWeight = opWeights.read || 0;
      const updateWeight = opWeights.update || 0;

      if (rand < createWeight) {
        operation = "create";
      } else if (rand < createWeight + readWeight) {
        operation = "read";
      } else if (rand < createWeight + readWeight + updateWeight) {
        operation = "update";
      } else {
        operation = "delete";
      }

      const start = Date.now();
      let hadError = false;
      try {
        await this.executeOperation(collection, operation, testClient);
        const duration = Date.now() - start;
        this.metrics.recordRequest(operation, duration, true);
      } catch (error: any) {
        hadError = true;
        const duration = Date.now() - start;
        this.metrics.recordRequest(
          operation,
          duration,
          false,
          error.message,
        );
      }

      // Adaptive delay based on operation type and success
      // Failed operations get longer delay to reduce retry pressure
      const delay = hadError ? 50 : 10;
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }

  private async executeOperation(
    collection: any,
    operation: "create" | "read" | "update" | "delete",
    testClient: TestClient,
  ): Promise<void> {
    switch (operation) {
      case "create": {
        const docRef = collection.doc();
        await docRef.set({
          title: `Load test todo ${Date.now()}`,
          completed: false,
          createdAt: new Date().toISOString(),
        });
        testClient.documentIds.push(docRef.id);
        break;
      }
      case "read": {
        if (testClient.documentIds.length > 0) {
          const randomId =
            testClient.documentIds[
              Math.floor(Math.random() * testClient.documentIds.length)
            ];
          const docRef = collection.doc(randomId);
          await docRef.get();
        } else {
          // If no documents, create one first
          const docRef = collection.doc();
          await docRef.set({
            title: `Load test todo ${Date.now()}`,
            completed: false,
            createdAt: new Date().toISOString(),
          });
          testClient.documentIds.push(docRef.id);
        }
        break;
      }
      case "update": {
        if (testClient.documentIds.length > 0) {
          const randomId =
            testClient.documentIds[
              Math.floor(Math.random() * testClient.documentIds.length)
            ];
          const docRef = collection.doc(randomId);
          await docRef.update({ completed: true });
        }
        break;
      }
      case "delete": {
        if (testClient.documentIds.length > 0) {
          const randomId = testClient.documentIds.pop()!;
          const docRef = collection.doc(randomId);
          await docRef.delete();
        }
        break;
      }
    }
  }

  private printResults(): void {
    const stats = this.metrics.getStatistics();

    console.log("\n📊 Load Test Results");
    console.log("===================");
    console.log(`Total Requests: ${stats.totalRequests}`);
    console.log(`Successful: ${stats.successfulRequests}`);
    console.log(`Failed: ${stats.failedRequests}`);
    console.log(`Success Rate: ${stats.successRate.toFixed(2)}%`);
    console.log(`Total Duration: ${(stats.totalDuration / 1000).toFixed(2)}s`);
    console.log(`Requests/Second: ${stats.requestsPerSecond.toFixed(2)}`);
    console.log("\nLatency (ms):");
    console.log(`  Mean: ${stats.latency.mean.toFixed(2)}`);
    console.log(`  Median: ${stats.latency.median.toFixed(2)}`);
    console.log(`  P95: ${stats.latency.p95.toFixed(2)}`);
    console.log(`  P99: ${stats.latency.p99.toFixed(2)}`);
    console.log(`  Min: ${stats.latency.min.toFixed(2)}`);
    console.log(`  Max: ${stats.latency.max.toFixed(2)}`);
    console.log("\nEvent Delivery:");
    console.log(`  Total Events: ${stats.eventDelivery.totalEvents}`);
    console.log(`  Avg Latency: ${stats.eventDelivery.avgLatency.toFixed(2)}ms`);
    
    // Operation breakdown
    const ops = stats.operationBreakdown || {};
    if (Object.keys(ops).length > 0) {
      console.log("\nOperation Breakdown:");
      for (const [op, count] of Object.entries(ops)) {
        console.log(`  ${op}: ${count}`);
      }
    }
    
    // Latency by operation type
    const latencies = stats.latencyByOperation || {};
    if (Object.keys(latencies).length > 0) {
      console.log("\nLatency by Operation (ms):");
      for (const [op, lat] of Object.entries(latencies)) {
        console.log(`  ${op}: Mean=${lat.mean.toFixed(2)}, P95=${lat.p95.toFixed(2)}, P99=${lat.p99.toFixed(2)}`);
      }
    }
  }

  exportResults(format: "json" | "csv" = "json"): string {
    if (format === "csv") {
      return this.metrics.exportToCSV();
    }
    return this.metrics.exportToJSON();
  }
}
