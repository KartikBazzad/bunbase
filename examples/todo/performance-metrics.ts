/**
 * Performance Metrics Collection and Analysis
 */

export interface RequestMetric {
  timestamp: number;
  operation: "create" | "read" | "update" | "delete";
  duration: number;
  success: boolean;
  error?: string;
}

export interface EventMetric {
  timestamp: number;
  eventType: "INSERT" | "UPDATE" | "DELETE";
  deliveryLatency: number;
}

export interface PerformanceMetrics {
  totalRequests: number;
  successfulRequests: number;
  failedRequests: number;
  totalDuration: number;
  requestMetrics: RequestMetric[];
  eventMetrics: EventMetric[];
  startTime: number;
  endTime: number;
}

export class MetricsCollector {
  private metrics: PerformanceMetrics;

  constructor() {
    this.metrics = {
      totalRequests: 0,
      successfulRequests: 0,
      failedRequests: 0,
      totalDuration: 0,
      requestMetrics: [],
      eventMetrics: [],
      startTime: Date.now(),
      endTime: 0,
    };
  }

  recordRequest(
    operation: "create" | "read" | "update" | "delete",
    duration: number,
    success: boolean,
    error?: string,
  ): void {
    this.metrics.totalRequests++;
    if (success) {
      this.metrics.successfulRequests++;
    } else {
      this.metrics.failedRequests++;
    }

    this.metrics.requestMetrics.push({
      timestamp: Date.now(),
      operation,
      duration,
      success,
      error,
    });
  }

  recordEvent(
    eventType: "INSERT" | "UPDATE" | "DELETE",
    deliveryLatency: number,
  ): void {
    this.metrics.eventMetrics.push({
      timestamp: Date.now(),
      eventType,
      deliveryLatency,
    });
  }

  finish(): void {
    this.metrics.endTime = Date.now();
    this.metrics.totalDuration = this.metrics.endTime - this.metrics.startTime;
  }

  getMetrics(): PerformanceMetrics {
    return { ...this.metrics };
  }

  getStatistics() {
    const durations = this.metrics.requestMetrics.map((m) => m.duration);
    const sorted = [...durations].sort((a, b) => a - b);

    const percentile = (arr: number[], p: number): number => {
      const index = Math.ceil((p / 100) * arr.length) - 1;
      return arr[Math.max(0, index)];
    };

    // Group metrics by operation type
    const operationBreakdown: Record<string, number> = {};
    const latencyByOperation: Record<string, { mean: number; p95: number; p99: number }> = {};
    
    for (const metric of this.metrics.requestMetrics) {
      operationBreakdown[metric.operation] = (operationBreakdown[metric.operation] || 0) + 1;
    }

    // Calculate latency stats per operation
    for (const op of ["create", "read", "update", "delete"] as const) {
      const opMetrics = this.metrics.requestMetrics.filter((m) => m.operation === op);
      if (opMetrics.length > 0) {
        const opDurations = opMetrics.map((m) => m.duration).sort((a, b) => a - b);
        latencyByOperation[op] = {
          mean: opDurations.reduce((a, b) => a + b, 0) / opDurations.length,
          p95: percentile(opDurations, 95),
          p99: percentile(opDurations, 99),
        };
      }
    }

    return {
      totalRequests: this.metrics.totalRequests,
      successfulRequests: this.metrics.successfulRequests,
      failedRequests: this.metrics.failedRequests,
      successRate: (this.metrics.successfulRequests / this.metrics.totalRequests) * 100,
      totalDuration: this.metrics.totalDuration,
      requestsPerSecond: (this.metrics.totalRequests / (this.metrics.totalDuration / 1000)),
      latency: {
        mean: durations.reduce((a, b) => a + b, 0) / durations.length,
        median: percentile(sorted, 50),
        p95: percentile(sorted, 95),
        p99: percentile(sorted, 99),
        min: Math.min(...durations),
        max: Math.max(...durations),
      },
      eventDelivery: {
        totalEvents: this.metrics.eventMetrics.length,
        avgLatency: this.metrics.eventMetrics.length > 0
          ? this.metrics.eventMetrics.reduce((a, b) => a + b.deliveryLatency, 0) /
            this.metrics.eventMetrics.length
          : 0,
      },
      operationBreakdown,
      latencyByOperation,
    };
  }

  exportToJSON(): string {
    return JSON.stringify(this.getMetrics(), null, 2);
  }

  exportToCSV(): string {
    const headers = ["timestamp", "operation", "duration", "success", "error"];
    const rows = this.metrics.requestMetrics.map((m) => [
      m.timestamp,
      m.operation,
      m.duration,
      m.success,
      m.error || "",
    ]);

    return [headers.join(","), ...rows.map((r) => r.join(","))].join("\n");
  }
}
