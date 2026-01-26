import { createClient } from "@bunbase/js-sdk";

const key = "bunbase_pk_live_z1tfXt5iGukSLAVj5kr3Uh13PHMugI3u";

// Create client with API key - databaseId is resolved automatically from the API key
const client = createClient({
  apiKey: key,
  // databaseId is optional - defaults to the database resolved from API key
  // You can override it if needed: databaseId: "custom-db-id"
});

async function main() {
  try {
    // First, create a collection and document
    // Note: Collections need to be created first via the admin API or dashboard
    // For now, let's try to create a document (which will create the collection if it doesn't exist)

    console.log("Creating a todo document...");
    const newDoc = await client.database.create("todos", {
      title: "My first todo",
      completed: false,
      createdAt: new Date().toISOString(),
    });

    console.log("Created document:", newDoc);

    // Now get the document
    console.log("\nFetching the document...");
    const doc = await client.database.get("todos", newDoc.documentId);
    console.log("Fetched document:", doc);

    // Query all todos
    console.log("\nQuerying all todos...");
    const todos = await client.database.query("todos");
    console.log(
      "All todos:",
      todos.data.map((todo) => todo.data),
    );
  } catch (error: any) {
    console.error("Error:", error.message);
    if (error.code) {
      console.error("Error code:", error.code);
    }
    if (error.status) {
      console.error("HTTP status:", error.status);
    }
    if (error.details) {
      console.error("Error details:", error.details);
    }
  }
}

main();
