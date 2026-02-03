import { createClient } from "../dist"; // Import from built dist

// Config
const API_URL = "http://localhost:3001";
const PROJECT_ID = "00000000-0000-0000-0000-000000000002"; // From seed
const EMAIL = "test-user@example.com"; // User of the app, not developer
const PASSWORD = "password123";

async function main() {
  console.log("🚀 Starting SDK Verification...");

  const client = createClient(API_URL, "pk_test123");

  // 1. Auth Test
  console.log("\n🔐 Testing Auth...");
  try {
    // Try login first
    console.log("Attempting login...");
    const loginRes = await client.auth.signInWithPassword({
      email: EMAIL,
      password: PASSWORD,
    });
    console.log("Login success:", loginRes);
  } catch (err) {
    console.log("Login failed, attempting registration...");
    try {
      const regRes = await client.auth.signUp({
        email: EMAIL,
        password: PASSWORD,
        name: "Test User",
      });
      console.log("Registration success:", regRes);
    } catch (regErr) {
      console.error("Auth failed:", regErr);
      process.exit(1);
    }
  }

  // 2. Database Test
  console.log("\n💾 Testing Database...");
  const colName = "test_col_" + Date.now();
  const collection = client.db.collection(colName);

  try {
    // Create Document
    console.log(`Creating document in '${colName}'...`);
    const doc = await collection.create({ hello: "world", time: Date.now() });
    console.log("Created Doc:", doc);

    // List Documents
    console.log("Listing documents...");
    const list = await collection.list();
    console.log("List Docs:", list);

    // Extract ID from doc (assuming Bundoc returns struct with ID, or we check response)
    // Bundoc usually returns { id: "...", ... }
    const docId = (doc as any).id || (doc as any)._id;

    if (docId) {
      // Get Document
      console.log(`Getting document ${docId}...`);
      const fetched = await collection.get(docId);
      console.log("Fetched Doc:", fetched);

      // Delete Document
      console.log(`Deleting document ${docId}...`);
      await collection.delete(docId);
      console.log("Deleted Doc");
    }
  } catch (err) {
    console.error("Database test failed:", err);
    // Don't exit, try functions
  }

  // 3. Functions Test
  console.log("\n⚡ Testing Functions...");
  const funcName = "hello-world";
  try {
    console.log(`Invoking function '${funcName}'...`);
    // Note: This requires the function to be deployed on the backend.
    // If not deployed, this will 404.
    const res = await client.functions.invoke(funcName, { name: "SDK" });
    console.log("Function Response:", res);
  } catch (err) {
    console.error(
      "Function test failed (expected if function not deployed):",
      err,
    );
  }

  console.log("\n✅ Verification Complete");
}

main().catch(console.error);
