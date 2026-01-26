import { createClient } from "@bunbase/js-sdk";
import * as readline from "readline";

const key = process.env.BUNBASE_API_KEY || "bunbase_pk_live_cxaZTm1TIJ9eQZYOUm2QVF48YuiTkufr";

// Create client with API key
const client = createClient({
  apiKey: key,
  baseURL: process.env.BUNBASE_URL || "http://localhost:3000/api",
});

const bunstore = client.bunstore();
const todosCollection = bunstore.collection("todos");

// Create readline interface for interactive CLI
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

function question(prompt: string): Promise<string> {
  return new Promise((resolve) => {
    rl.question(prompt, resolve);
  });
}

function printMenu() {
  console.log("\n=== BunStore Todo Test Suite ===");
  console.log("1. Create todo");
  console.log("2. List todos (with filters)");
  console.log("3. Update todo");
  console.log("4. Delete todo");
  console.log("5. Subscribe to real-time updates");
  console.log("6. Batch operations");
  console.log("7. Transaction example");
  console.log("8. Query examples");
  console.log("9. Performance test");
  console.log("0. Exit");
  console.log("================================");
}

async function createTodo() {
  try {
    const title = await question("Enter todo title: ");
    const completed = await question("Is it completed? (y/n): ");
    
    const docRef = todosCollection.doc();
    await docRef.set({
      title,
      completed: completed.toLowerCase() === "y",
      createdAt: new Date().toISOString(),
    });

    console.log(`✅ Todo created with ID: ${docRef.id}`);
  } catch (error: any) {
    console.error("❌ Error creating todo:", error.message);
  }
}

async function listTodos() {
  try {
    const filterOption = await question(
      "Filter by completed? (y/n/all): "
    );
    
    let query = todosCollection;
    
    if (filterOption.toLowerCase() === "y") {
      query = query.where("completed", "==", true);
    } else if (filterOption.toLowerCase() === "n") {
      query = query.where("completed", "==", false);
    }
    
    const sortOption = await question("Sort by? (createdAt/title/none): ");
    if (sortOption === "createdAt") {
      query = query.orderBy("createdAt", "desc");
    } else if (sortOption === "title") {
      query = query.orderBy("title", "asc");
    }
    
    const limitStr = await question("Limit (default 10): ");
    const limit = limitStr ? parseInt(limitStr) : 10;
    query = query.limit(limit);
    
    const snapshot = await query.get();
    
    console.log(`\n📋 Found ${snapshot.size} todos:`);
    snapshot.docs.forEach((doc, index) => {
      const data = doc.data();
      console.log(
        `${index + 1}. [${doc.id}] ${data?.title || "Untitled"} - ${
          data?.completed ? "✅" : "⏳"
        }`
      );
    });
  } catch (error: any) {
    console.error("❌ Error listing todos:", error.message);
  }
}

async function updateTodo() {
  try {
    const todoId = await question("Enter todo ID: ");
    const docRef = todosCollection.doc(todoId);
    
    const currentDoc = await docRef.get();
    if (!currentDoc.exists()) {
      console.error("❌ Todo not found");
      return;
    }
    
    console.log("Current todo:", currentDoc.data());
    
    const field = await question("Field to update (title/completed): ");
    let value: any = await question("New value: ");
    
    if (field === "completed") {
      value = value.toLowerCase() === "y" || value === "true";
    }
    
    await docRef.update({ [field]: value });
    console.log("✅ Todo updated");
  } catch (error: any) {
    console.error("❌ Error updating todo:", error.message);
  }
}

async function deleteTodo() {
  try {
    const todoId = await question("Enter todo ID to delete: ");
    const docRef = todosCollection.doc(todoId);
    
    await docRef.delete();
    console.log("✅ Todo deleted");
  } catch (error: any) {
    console.error("❌ Error deleting todo:", error.message);
  }
}

async function subscribeRealtime() {
  try {
    console.log("🔔 Subscribing to real-time updates...");
    console.log("(Press Enter to stop)");
    
    const unsubscribe = todosCollection.onSnapshot((snapshot) => {
      console.log(`\n📡 Real-time update: ${snapshot.size} todos`);
      snapshot.docs.forEach((doc) => {
        const data = doc.data();
        console.log(`  - [${doc.id}] ${data?.title || "Untitled"}`);
      });
    });
    
    await question("");
    unsubscribe();
    console.log("✅ Unsubscribed");
  } catch (error: any) {
    console.error("❌ Error subscribing:", error.message);
  }
}

async function batchOperations() {
  try {
    console.log("📦 Batch operations test");
    const count = parseInt(await question("How many todos to create? ")) || 5;
    
    const batch = bunstore.batch();
    const docRefs: any[] = [];
    
    for (let i = 0; i < count; i++) {
      const docRef = todosCollection.doc();
      batch.set(docRef, {
        title: `Batch todo ${i + 1}`,
        completed: false,
        createdAt: new Date().toISOString(),
      });
      docRefs.push(docRef);
    }
    
    await batch.commit();
    console.log(`✅ Created ${count} todos in batch`);
  } catch (error: any) {
    console.error("❌ Error in batch operations:", error.message);
  }
}

async function transactionExample() {
  try {
    console.log("🔄 Transaction example");
    const todoId1 = await question("Enter first todo ID: ");
    const todoId2 = await question("Enter second todo ID: ");
    
    await bunstore.runTransaction(async (transaction) => {
      const doc1Ref = todosCollection.doc(todoId1);
      const doc2Ref = todosCollection.doc(todoId2);
      
      const doc1 = await transaction.get(doc1Ref);
      const doc2 = await transaction.get(doc2Ref);
      
      if (doc1.exists() && doc2.exists()) {
        transaction.update(doc1Ref, { completed: true });
        transaction.update(doc2Ref, { completed: true });
        console.log("✅ Transaction will mark both todos as completed");
      } else {
        throw new Error("One or both todos not found");
      }
    });
    
    console.log("✅ Transaction committed");
  } catch (error: any) {
    console.error("❌ Error in transaction:", error.message);
  }
}

async function queryExamples() {
  try {
    console.log("🔍 Query examples");
    
    // Example 1: All incomplete todos
    console.log("\n1. All incomplete todos:");
    const incomplete = await todosCollection
      .where("completed", "==", false)
      .get();
    incomplete.docs.forEach((doc) => {
      console.log(`   - ${doc.data()?.title}`);
    });
    
    // Example 2: Limit and order
    console.log("\n2. Latest 3 todos:");
    const latest = await todosCollection
      .orderBy("createdAt", "desc")
      .limit(3)
      .get();
    latest.docs.forEach((doc) => {
      console.log(`   - ${doc.data()?.title}`);
    });
    
    // Example 3: Offset pagination
    console.log("\n3. Todos with offset (skip first 2):");
    const paginated = await todosCollection
      .offset(2)
      .limit(5)
      .get();
    paginated.docs.forEach((doc) => {
      console.log(`   - ${doc.data()?.title}`);
    });
  } catch (error: any) {
    console.error("❌ Error in query examples:", error.message);
  }
}

async function performanceTest() {
  try {
    console.log("⚡ Performance test");
    const count = parseInt(await question("How many operations? ")) || 100;
    
    console.log(`\nCreating ${count} todos...`);
    const startTime = Date.now();
    
    const promises = [];
    for (let i = 0; i < count; i++) {
      const docRef = todosCollection.doc();
      promises.push(
        docRef.set({
          title: `Perf test todo ${i}`,
          completed: false,
          createdAt: new Date().toISOString(),
        })
      );
    }
    
    await Promise.all(promises);
    const endTime = Date.now();
    const duration = endTime - startTime;
    
    console.log(`✅ Created ${count} todos in ${duration}ms`);
    console.log(`   Average: ${(duration / count).toFixed(2)}ms per operation`);
    console.log(`   Throughput: ${((count / duration) * 1000).toFixed(2)} ops/sec`);
  } catch (error: any) {
    console.error("❌ Error in performance test:", error.message);
  }
}

async function main() {
  console.log("Welcome to BunStore Todo Test Suite!");
  console.log("This is a comprehensive testing ground for BunStore features.");
  
  while (true) {
    printMenu();
    const choice = await question("\nSelect an option: ");
    
    switch (choice) {
      case "1":
        await createTodo();
        break;
      case "2":
        await listTodos();
        break;
      case "3":
        await updateTodo();
        break;
      case "4":
        await deleteTodo();
        break;
      case "5":
        await subscribeRealtime();
        break;
      case "6":
        await batchOperations();
        break;
      case "7":
        await transactionExample();
        break;
      case "8":
        await queryExamples();
        break;
      case "9":
        await performanceTest();
        break;
      case "0":
        console.log("👋 Goodbye!");
        rl.close();
        process.exit(0);
      default:
        console.log("❌ Invalid option");
    }
  }
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
