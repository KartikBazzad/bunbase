import { DocDBClient } from '../src';

const client = new DocDBClient({
  socketPath: '/tmp/docdb.sock'
});

async function main() {
  try {
    console.log('=== DocDB TypeScript Example ===');
    console.log();

    // Connect to server
    await client.connect();
    console.log('Connected to DocDB server');
    console.log();

    // Open database
    const dbName = `example_db_${Date.now()}`;
    console.log(`Creating database: ${dbName}`);
    const dbID = await client.openDB(dbName);
    console.log(`Created database with ID: ${dbID}`);
    console.log();

    // Create documents
    console.log('Creating documents...');
    for (let i = 1; i <= 5; i++) {
      const payload = new TextEncoder().encode(`Document ${i} payload`);
      await client.create(dbID, BigInt(i), payload);
      console.log(`  Created document ${i}`);
    }
    console.log();

    // Read documents
    console.log('Reading documents...');
    for (let i = 1; i <= 5; i++) {
      const data = await client.read(dbID, BigInt(i));
      console.log(`  Document ${i}: ${new TextDecoder().decode(data)}`);
    }
    console.log();

    // Update document
    console.log('Updating document 3...');
    const newPayload = new TextEncoder().encode('Updated payload for document 3');
    await client.update(dbID, 3n, newPayload);
    console.log('  Document 3 updated');
    console.log();

    // Read updated document
    console.log('Reading updated document 3...');
    const data = await client.read(dbID, 3n);
    console.log(`  Document 3: ${new TextDecoder().decode(data)}`);
    console.log();

    // Delete document
    console.log('Deleting document 5...');
    await client.delete(dbID, 5n);
    console.log('  Document 5 deleted');
    console.log();

    // Read documents after delete
    console.log('Reading documents after delete...');
    for (let i = 1; i <= 5; i++) {
      try {
        const docData = await client.read(dbID, BigInt(i));
        console.log(`  Document ${i}: ${new TextDecoder().decode(docData)}`);
      } catch (e: any) {
        if (e.code === 2) {
          console.log(`  Document ${i}: not found (as expected)`);
        } else {
          console.log(`  Document ${i}: error - ${e.message}`);
        }
      }
    }
    console.log();

    // Get stats
    console.log('Getting pool stats...');
    const stats = await client.stats();
    console.log(`  Total DBs: ${stats.totalDBs}`);
    console.log(`  Active DBs: ${stats.activeDBs}`);
    console.log(`  Memory Used: ${stats.memoryUsed} bytes`);
    console.log();

    // Close database
    console.log('Closing database...');
    await client.closeDB(dbID);
    console.log('  Database closed');
    console.log();

    // Disconnect
    await client.disconnect();
    console.log('Disconnected');

    console.log();
    console.log('=== Example Complete ===');
  } catch (error: any) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

main();
