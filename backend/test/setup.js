/**
 * Test setup and teardown
 * Runs before and after all tests
 */

import { beforeAll, afterAll } from 'vitest';
import mongoose from 'mongoose';
import { MongoMemoryServer } from 'mongodb-memory-server';

let mongoServer;

beforeAll(async () => {
  // Setup in-memory MongoDB for testing
  mongoServer = await MongoMemoryServer.create();
  const mongoUri = mongoServer.getUri();
  
  await mongoose.connect(mongoUri);
  console.log('Test MongoDB connected');
}, 30000); // 30 second timeout for setup

afterAll(async () => {
  // Close MongoDB connection
  if (mongoose.connection.readyState !== 0) {
    await mongoose.connection.dropDatabase();
    await mongoose.connection.close();
  }

  // Stop MongoDB memory server
  if (mongoServer) {
    await mongoServer.stop();
  }

  console.log('Test environment cleaned up');
}, 30000); // 30 second timeout for teardown

