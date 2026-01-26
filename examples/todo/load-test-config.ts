/**
 * Load Test Configuration
 * 
 * Defines test profiles and customizable parameters for load testing scenarios.
 */

export interface LoadTestConfig {
  name: string;
  description: string;
  concurrentUsers: number;
  requestRate?: number; // requests per second per user
  duration: number; // milliseconds
  collectionName: string;
  documentCount?: number;
  operationMix: {
    create: number;
    read: number;
    update: number;
    delete: number;
  };
}

export const testProfiles: Record<string, LoadTestConfig> = {
  light: {
    name: "Light Load",
    description: "Low traffic test",
    concurrentUsers: 10,
    duration: 30000, // 30 seconds
    collectionName: "load_test",
    operationMix: {
      create: 40,
      read: 30,
      update: 20,
      delete: 10,
    },
  },
  medium: {
    name: "Medium Load",
    description: "Moderate traffic test",
    concurrentUsers: 100,
    duration: 60000, // 1 minute
    collectionName: "load_test",
    operationMix: {
      create: 40,
      read: 30,
      update: 20,
      delete: 10,
    },
  },
  heavy: {
    name: "Heavy Load",
    description: "High traffic test",
    concurrentUsers: 500,
    duration: 120000, // 2 minutes
    collectionName: "load_test",
    operationMix: {
      create: 40,
      read: 30,
      update: 20,
      delete: 10,
    },
  },
  extreme: {
    name: "Extreme Load",
    description: "Very high traffic test",
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
  readHeavy: {
    name: "Read-Heavy",
    description: "80% reads, 20% writes",
    concurrentUsers: 200,
    duration: 60000,
    collectionName: "load_test",
    operationMix: {
      create: 10,
      read: 80,
      update: 8,
      delete: 2,
    },
  },
  writeHeavy: {
    name: "Write-Heavy",
    description: "80% writes, 20% reads",
    concurrentUsers: 200,
    duration: 60000,
    collectionName: "load_test",
    operationMix: {
      create: 50,
      read: 20,
      update: 25,
      delete: 5,
    },
  },
};

export function getConfig(profile: string): LoadTestConfig | null {
  return testProfiles[profile] || null;
}

export function listProfiles(): string[] {
  return Object.keys(testProfiles);
}
