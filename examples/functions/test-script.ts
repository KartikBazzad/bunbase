/**
 * Test Script for Functions
 * Creates and tests example functions
 */

import { readFileSync } from "fs";
import { join } from "path";

// You'll need to set these based on your setup
const API_BASE_URL = process.env.API_BASE_URL || "http://localhost:3000/api";
const API_KEY = process.env.API_KEY || ""; // Set your API key here

interface FunctionExample {
  name: string;
  file: string;
  description: string;
}

const examples: FunctionExample[] = [
  {
    name: "hello-world",
    file: "hello-world.ts",
    description: "Simple greeting function",
  },
  {
    name: "echo",
    file: "echo.ts",
    description: "Echoes request data",
  },
  {
    name: "json-processor",
    file: "json-processor.ts",
    description: "Processes JSON data",
  },
  {
    name: "calculator",
    file: "calculator.ts",
    description: "Simple calculator API",
  },
  {
    name: "env-demo",
    file: "env-demo.ts",
    description: "Environment variables demo",
  },
];

async function createFunction(
  name: string,
  code: string,
): Promise<string | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/functions`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-API-Key": API_KEY,
      },
      body: JSON.stringify({
        name,
        runtime: "bun",
        handler: "handler",
        code,
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      console.error(`Failed to create ${name}:`, error);
      return null;
    }

    const data = await response.json();
    console.log(`✅ Created function: ${name} (ID: ${data.id})`);
    return data.id;
  } catch (error: any) {
    console.error(`Error creating ${name}:`, error.message);
    return null;
  }
}

async function deployFunction(functionId: string): Promise<boolean> {
  try {
    const response = await fetch(
      `${API_BASE_URL}/functions/${functionId}/deploy`,
      {
        method: "POST",
        headers: {
          "X-API-Key": API_KEY,
        },
      },
    );

    if (!response.ok) {
      const error = await response.json();
      console.error(`Failed to deploy ${functionId}:`, error);
      return false;
    }

    const data = await response.json();
    console.log(`✅ Deployed function (version: ${data.version})`);
    return true;
  } catch (error: any) {
    console.error(`Error deploying ${functionId}:`, error.message);
    return false;
  }
}

async function invokeFunction(
  functionName: string,
  method: string = "GET",
  body?: any,
): Promise<void> {
  try {
    const url = `${API_BASE_URL}/functions/${functionName}`;
    const options: RequestInit = {
      method,
      headers: {
        "X-API-Key": API_KEY,
        ...(body && { "Content-Type": "application/json" }),
      },
      ...(body && { body: JSON.stringify(body) }),
    };

    const response = await fetch(url, options);
    const data = await response.json();

    console.log(`\n📤 Invoked ${functionName}:`);
    console.log(`   Status: ${response.status}`);
    console.log(`   Response:`, JSON.stringify(data, null, 2));
  } catch (error: any) {
    console.error(`Error invoking ${functionName}:`, error.message);
  }
}

async function main() {
  if (!API_KEY) {
    console.error(
      "❌ API_KEY not set. Set it as an environment variable or in this script.",
    );
    console.log("\nUsage:");
    console.log("  API_KEY=your_key bun run test-script.ts");
    process.exit(1);
  }

  console.log("🚀 Creating test functions...\n");

  const functionIds: Array<{ name: string; id: string }> = [];

  // Create all functions
  for (const example of examples) {
    const codePath = join(import.meta.dir, example.file);
    const code = readFileSync(codePath, "utf-8");

    const functionId = await createFunction(example.name, code);
    if (functionId) {
      functionIds.push({ name: example.name, id: functionId });
    }
  }

  console.log("\n📦 Deploying functions...\n");

  // Deploy all functions
  for (const { name, id } of functionIds) {
    await deployFunction(id);
  }

  console.log("\n🧪 Testing functions...\n");

  // Test hello-world
  await invokeFunction("hello-world?name=TestUser");

  // Test echo with POST
  await invokeFunction("echo", "POST", { test: "data", number: 42 });

  // Test calculator
  await invokeFunction("calculator?a=15&b=3&op=multiply");

  // Test json-processor
  await invokeFunction("json-processor", "POST", [1, 2, 3, 4, 5]);

  // Test env-demo
  await invokeFunction("env-demo");

  console.log("\n✅ All tests completed!");
  console.log("\nYou can now invoke functions via:");
  console.log(`  GET/POST http://localhost:3000/api/functions/{function-name}`);
  console.log(`  Or via: http://localhost:3000/api/functions/{function-id}/invoke`);
}

main().catch(console.error);
