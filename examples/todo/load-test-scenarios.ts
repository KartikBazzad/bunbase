/**
 * Load Test Scenarios
 * 
 * Predefined test scenarios for various load testing patterns.
 */

import type { LoadTestConfig } from "./load-test-config";

export interface LoadTestScenario {
  name: string;
  description: string;
  config: LoadTestConfig;
}

export const scenarios: Record<string, LoadTestScenario> = {
  crud: {
    name: "CRUD Load Test",
    description: "Mixed CRUD operations (40% create, 30% read, 20% update, 10% delete)",
    config: {
      name: "CRUD Load Test",
      description: "Mixed CRUD operations",
      concurrentUsers: 1000,
      duration: 300000, // 5 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  realtime: {
    name: "Real-time Subscription Test",
    description: "Multiple clients subscribing to document changes",
    config: {
      name: "Real-time Subscription Test",
      description: "Test real-time event delivery",
      concurrentUsers: 500, // subscribers
      duration: 600000, // 10 minutes
      collectionName: "load_test",
      documentCount: 100, // writers
      operationMix: {
        create: 100, // Only creates for this test
        read: 0,
        update: 0,
        delete: 0,
      },
    },
  },
  batch: {
    name: "Batch Operations Test",
    description: "Large batch creates/updates/deletes",
    config: {
      name: "Batch Operations Test",
      description: "Test batch operation performance",
      concurrentUsers: 100,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 50,
        read: 0,
        update: 30,
        delete: 20,
      },
    },
  },
  stress: {
    name: "Stress Test",
    description: "Gradual ramp-up to find breaking point",
    config: {
      name: "Stress Test",
      description: "Gradual ramp-up",
      concurrentUsers: 2000, // Will ramp up gradually
      duration: 600000, // 10 minutes (2 min per stage)
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  spike: {
    name: "Spike Test",
    description: "Sudden traffic spike simulation",
    config: {
      name: "Spike Test",
      description: "Sudden traffic spike",
      concurrentUsers: 2000,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  // Scaled user count scenarios (100-1000 users)
  users100: {
    name: "100 Users Test",
    description: "CRUD operations with 100 concurrent users",
    config: {
      name: "100 Users Test",
      description: "100 concurrent users - CRUD operations",
      concurrentUsers: 100,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users200: {
    name: "200 Users Test",
    description: "CRUD operations with 200 concurrent users",
    config: {
      name: "200 Users Test",
      description: "200 concurrent users - CRUD operations",
      concurrentUsers: 200,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users300: {
    name: "300 Users Test",
    description: "CRUD operations with 300 concurrent users",
    config: {
      name: "300 Users Test",
      description: "300 concurrent users - CRUD operations",
      concurrentUsers: 300,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users400: {
    name: "400 Users Test",
    description: "CRUD operations with 400 concurrent users",
    config: {
      name: "400 Users Test",
      description: "400 concurrent users - CRUD operations",
      concurrentUsers: 400,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users500: {
    name: "500 Users Test",
    description: "CRUD operations with 500 concurrent users",
    config: {
      name: "500 Users Test",
      description: "500 concurrent users - CRUD operations",
      concurrentUsers: 500,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users600: {
    name: "600 Users Test",
    description: "CRUD operations with 600 concurrent users",
    config: {
      name: "600 Users Test",
      description: "600 concurrent users - CRUD operations",
      concurrentUsers: 600,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users700: {
    name: "700 Users Test",
    description: "CRUD operations with 700 concurrent users",
    config: {
      name: "700 Users Test",
      description: "700 concurrent users - CRUD operations",
      concurrentUsers: 700,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users800: {
    name: "800 Users Test",
    description: "CRUD operations with 800 concurrent users",
    config: {
      name: "800 Users Test",
      description: "800 concurrent users - CRUD operations",
      concurrentUsers: 800,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users900: {
    name: "900 Users Test",
    description: "CRUD operations with 900 concurrent users",
    config: {
      name: "900 Users Test",
      description: "900 concurrent users - CRUD operations",
      concurrentUsers: 900,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
  users1000: {
    name: "1000 Users Test",
    description: "CRUD operations with 1000 concurrent users",
    config: {
      name: "1000 Users Test",
      description: "1000 concurrent users - CRUD operations",
      concurrentUsers: 1000,
      duration: 180000, // 3 minutes
      collectionName: "load_test",
      operationMix: {
        create: 40,
        read: 30,
        update: 20,
        delete: 10,
      },
    },
  },
};

export function getScenario(name: string): LoadTestScenario | null {
  return scenarios[name] || null;
}

export function listScenarios(): string[] {
  return Object.keys(scenarios);
}
