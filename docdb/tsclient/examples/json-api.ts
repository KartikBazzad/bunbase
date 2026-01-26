import { DocDBJSONClient } from '../src';

interface User {
  id: number;
  name: string;
  email: string;
  createdAt: Date;
}

const client = new DocDBJSONClient();

async function main() {
  try {
    console.log('=== DocDB JSON API Example ===');
    console.log();

    // Connect to server
    await client.connect();
    console.log('Connected to DocDB server');
    console.log();

    // Open database
    const dbName = `usersdb_${Date.now()}`;
    console.log(`Creating database: ${dbName}`);
    const dbID = await client.openDB(dbName);
    console.log(`Created database with ID: ${dbID}`);
    console.log();

    // Create JSON documents
    console.log('Creating user documents...');
    const users: User[] = [
      {
        id: 1,
        name: 'John Doe',
        email: 'john@example.com',
        createdAt: new Date('2024-01-01')
      },
      {
        id: 2,
        name: 'Jane Smith',
        email: 'jane@example.com',
        createdAt: new Date('2024-01-02')
      },
      {
        id: 3,
        name: 'Bob Johnson',
        email: 'bob@example.com',
        createdAt: new Date('2024-01-03')
      }
    ];

    for (const user of users) {
      await client.createJSON<User>(dbID, BigInt(user.id), user);
      console.log(`  Created user: ${user.name} (${user.email})`);
    }
    console.log();

    // Read JSON documents
    console.log('Reading user documents...');
    for (const user of users) {
      const fetched = await client.readJSON<User>(dbID, BigInt(user.id));
      if (fetched) {
        console.log(`  User ${fetched.id}: ${fetched.name} - ${fetched.email}`);
      } else {
        console.log(`  User ${user.id}: not found`);
      }
    }
    console.log();

    // Update JSON document
    console.log('Updating user 2...');
    const updatedUser: User = {
      ...users[1],
      name: 'Jane Smith-Updated',
      email: 'jane.smith@example.com'
    };
    await client.updateJSON<User>(dbID, 2n, updatedUser);
    console.log(`  Updated to: ${updatedUser.name} - ${updatedUser.email}`);
    console.log();

    // Verify update
    console.log('Verifying update...');
    const verified = await client.readJSON<User>(dbID, 2n);
    if (verified) {
      console.log(`  Verified: ${verified.name} - ${verified.email}`);
    }
    console.log();

    // Delete JSON document
    console.log('Deleting user 3...');
    await client.delete(dbID, 3n);
    console.log('  User 3 deleted');
    console.log();

    // Verify deletion
    console.log('Verifying deletion...');
    const deleted = await client.readJSON<User>(dbID, 3n);
    if (deleted === null) {
      console.log('  Verified: user 3 not found (as expected)');
    } else {
      console.log('  ERROR: user 3 still exists!');
    }
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
    console.log('=== JSON API Example Complete ===');
  } catch (error: any) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

main();
