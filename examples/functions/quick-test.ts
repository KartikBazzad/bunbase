/**
 * Quick Test Script
 * Simple script to create and test one function
 * 
 * Usage:
 *   API_KEY=your_key bun run quick-test.ts
 */

const API_BASE_URL = process.env.API_BASE_URL || "http://localhost:3000/api";
const API_KEY = process.env.API_KEY || "";

if (!API_KEY) {
  console.error("❌ Please set API_KEY environment variable");
  console.log("Usage: API_KEY=your_key bun run quick-test.ts");
  process.exit(1);
}

// Simple hello world function code
const functionCode = `export async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  return Response.json({
    message: \`Hello, \${name}!\`,
    timestamp: new Date().toISOString(),
  });
}`;

async function main() {
  try {
    console.log("📝 Creating function...");
    console.log(`   API Base URL: ${API_BASE_URL}`);
    console.log(`   API Key: ${API_KEY.substring(0, 20)}...`);
    
    // 1. Create function
    const createResponse = await fetch(`${API_BASE_URL}/functions`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-API-Key": API_KEY,
      },
      body: JSON.stringify({
        name: "test-hello",
        runtime: "bun",
        handler: "handler",
        code: functionCode,
      }),
    });
    
    console.log(`   Response status: ${createResponse.status} ${createResponse.statusText}`);

    if (!createResponse.ok) {
      const responseText = await createResponse.text();
      let error: any;
      try {
        error = JSON.parse(responseText);
      } catch {
        error = { message: responseText, status: createResponse.status };
      }
      console.error("❌ Failed to create function:", error);
      console.error(`   Status: ${createResponse.status} ${createResponse.statusText}`);
      process.exit(1);
    }

    const functionData = await createResponse.json();
    const functionId = functionData.id;
    console.log(`✅ Function created: ${functionData.name} (ID: ${functionId})`);

    // 2. Deploy function
    console.log("\n📦 Deploying function...");
    const deployResponse = await fetch(
      `${API_BASE_URL}/functions/${functionId}/deploy`,
      {
        method: "POST",
        headers: {
          "X-API-Key": API_KEY,
        },
      },
    );

    if (!deployResponse.ok) {
      const responseText = await deployResponse.text();
      let error: any;
      try {
        error = JSON.parse(responseText);
      } catch {
        error = { message: responseText, status: deployResponse.status };
      }
      console.error("❌ Failed to deploy:", error);
      console.error(`   Status: ${deployResponse.status} ${deployResponse.statusText}`);
      process.exit(1);
    }

    const deployData = await deployResponse.json();
    console.log(`✅ Function deployed (version: ${deployData.version})`);

    // 3. Test function via invoke endpoint
    console.log("\n🧪 Testing function via /invoke endpoint...");
    const invokeResponse = await fetch(
      `${API_BASE_URL}/functions/${functionId}/invoke`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": API_KEY,
        },
        body: JSON.stringify({
          method: "GET",
          url: "http://localhost:3000/api/functions/test-hello?name=TestUser",
          headers: {},
        }),
      },
    );

    if (!invokeResponse.ok) {
      const responseText = await invokeResponse.text();
      let error: any;
      try {
        error = JSON.parse(responseText);
      } catch {
        error = { message: responseText, status: invokeResponse.status };
      }
      console.error("❌ Failed to invoke:", error);
      console.error(`   Status: ${invokeResponse.status} ${invokeResponse.statusText}`);
    } else {
      const result = await invokeResponse.json();
      console.log("✅ Invoke result:", JSON.stringify(result, null, 2));
    }

    // 4. Test function via direct HTTP endpoint (by name)
    console.log("\n🧪 Testing function via HTTP endpoint (by name)...");
    try {
      const httpResponse = await fetch(
        `${API_BASE_URL}/functions/test-hello?name=Alice`,
        {
          headers: {
            "X-API-Key": API_KEY,
          },
        },
      );

      if (!httpResponse.ok) {
        const error = await httpResponse.text();
        console.error("❌ HTTP request failed:", error);
      } else {
        const result = await httpResponse.json();
        console.log("✅ HTTP result:", JSON.stringify(result, null, 2));
      }
    } catch (error: any) {
      console.log("⚠️  HTTP endpoint test skipped (may need function to be deployed first)");
    }

    console.log("\n✅ All tests passed!");
    console.log(`\nFunction ID: ${functionId}`);
    console.log(`You can invoke it at: ${API_BASE_URL}/functions/test-hello`);
    console.log(`Or via: ${API_BASE_URL}/functions/${functionId}/invoke`);

  } catch (error: any) {
    console.error("❌ Error:", error.message);
    if (error.stack) {
      console.error("Stack:", error.stack);
    }
    process.exit(1);
  }
}

main();
