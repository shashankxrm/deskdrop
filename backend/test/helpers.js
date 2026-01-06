/**
 * Test helper functions and utilities
 */

import { Device } from '../src/models/Device.js';
import { Link } from '../src/models/Link.js';
import { User } from '../src/models/User.js';

/**
 * Create a test user
 */
export async function createTestUser(userId = 'test-user-' + Date.now()) {
  const user = new User({
    userId,
    email: `test-${userId}@example.com`,
  });
  await user.save();
  return user;
}

/**
 * Create a test device
 */
export async function createTestDevice(options = {}) {
  const {
    userId = 'test-user-' + Date.now(),
    deviceId = 'test-device-' + Date.now(),
    deviceName = 'Test Device',
    deviceType = 'desktop',
    pairingToken = 'test-pairing-token-' + Date.now(),
    isOnline = false,
    socketId = null,
  } = options;

  const device = new Device({
    userId,
    deviceId,
    deviceName,
    deviceType,
    pairingToken: deviceType === 'desktop' ? pairingToken : null,
    isOnline,
    socketId,
  });
  await device.save();
  return device;
}

/**
 * Create a test link
 */
export async function createTestLink(options = {}) {
  const {
    userId = 'test-user-' + Date.now(),
    deviceId = 'test-device-' + Date.now(),
    url = 'https://example.com/test-' + Date.now(),
    status = 'pending',
  } = options;

  const link = new Link({
    userId,
    deviceId,
    url,
    status,
  });
  await link.save();
  return link;
}

/**
 * Clean up test data
 */
export async function cleanupTestData() {
  await Device.deleteMany({});
  await Link.deleteMany({});
  await User.deleteMany({});
}

/**
 * Mock Socket.IO instance for testing
 */
export function createMockSocketIO() {
  const sockets = new Map();
  const events = [];

  const mockSocket = {
    id: 'test-socket-' + Date.now(),
    emit: (event, data) => {
      events.push({ event, data, socketId: mockSocket.id });
    },
    disconnect: () => {},
  };

  const mockIO = {
    sockets: {
      sockets: {
        get: (socketId) => sockets.get(socketId) || null,
        set: (socketId, socket) => sockets.set(socketId, socket),
      },
    },
    on: (event, handler) => {},
    use: (middleware) => {},
    emit: (event, data) => {
      events.push({ event, data, broadcast: true });
    },
  };

  // Helper to add a socket
  mockIO.addSocket = (socketId, socket = mockSocket) => {
    sockets.set(socketId, socket);
    return socket;
  };

  // Helper to get emitted events
  mockIO.getEmittedEvents = () => events;
  mockIO.clearEvents = () => events.length = 0;

  return { mockIO, mockSocket };
}

