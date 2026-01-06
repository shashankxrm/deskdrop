import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import express from 'express';
import request from 'supertest';
import linksRouter from '../../src/routes/links.js';
import { cleanupTestData, createTestDevice } from '../helpers.js';

// Mock the auth middleware
vi.mock('../../src/middleware/auth.js', () => ({
  authenticate: (req, res, next) => {
    // Set test user and device for all requests
    req.userId = 'test-user-123';
    req.deviceId = 'test-device-123';
    next();
  },
  authenticateDevToken: (req, res, next) => {
    req.userId = 'test-user-123';
    req.deviceId = 'test-device-123';
    next();
  },
  authenticateWebAuthn: (req, res, next) => {
    req.userId = 'test-user-123';
    req.email = 'test@example.com';
    next();
  },
}));

// Mock Redis service
const mockRedisStore = new Map();
vi.mock('../../src/services/redisService.js', () => ({
  queueLinkForDevice: vi.fn(async (deviceId, linkData) => {
    const queueKey = `queue:device:${deviceId}`;
    const existing = mockRedisStore.get(queueKey) || [];
    existing.push(linkData);
    mockRedisStore.set(queueKey, existing);
  }),
  getQueuedLinksForDevice: vi.fn(async (deviceId) => {
    const queueKey = `queue:device:${deviceId}`;
    return mockRedisStore.get(queueKey) || [];
  }),
  clearQueueForDevice: vi.fn(async (deviceId) => {
    const queueKey = `queue:device:${deviceId}`;
    mockRedisStore.delete(queueKey);
  }),
}));

// Mock Socket.IO
const mockIO = {
  sockets: {
    sockets: {
      get: () => null,
    },
  },
};

describe('POST /api/links', () => {
  let app;

  beforeEach(async () => {
    // Wait for MongoDB to be ready
    const mongoose = await import('mongoose');
    if (mongoose.default.connection.readyState === 0) {
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    await cleanupTestData();
    mockRedisStore.clear();
    vi.clearAllMocks();
    app = express();
    app.use(express.json());
    app.use('/api/links', linksRouter);
    app.set('io', mockIO);
  });

  afterEach(async () => {
    await cleanupTestData();
  });

  it('should return 400 if URL is missing', async () => {
    const response = await request(app)
      .post('/api/links')
      .send({ deviceId: 'test-device' });

    expect(response.status).toBe(400);
    expect(response.body.error).toBe('URL is required');
  });

  it('should return 400 if URL is invalid format', async () => {
    const response = await request(app)
      .post('/api/links')
      .send({ url: 'not-a-url', deviceId: 'test-device' });

    expect(response.status).toBe(400);
    expect(response.body.error).toBe('Invalid URL format');
  });

  it('should successfully queue link when no desktop device exists', async () => {
    const response = await request(app)
      .post('/api/links')
      .send({ 
        url: 'https://example.com/test',
        deviceId: 'android-device-123'
      });

    expect(response.status).toBe(200);
    expect(response.body.success).toBe(true);
    expect(response.body.queued).toBe(true);

    // Verify link was saved to database
    const { Link } = await import('../../src/models/Link.js');
    const link = await Link.findOne({ url: 'https://example.com/test' });
    expect(link).toBeTruthy();
    expect(link.status).toBe('pending');
  });

  it('should successfully deliver link when desktop device is online', async () => {
    const userId = 'test-user-123';
    const socketId = 'socket-123';
    
    // Create online desktop device
    const desktopDevice = await createTestDevice({
      userId,
      deviceType: 'desktop',
      isOnline: true,
      socketId,
      pairingToken: 'pairing-token-1',
    });

    // Mock socket in IO
    const mockSocket = {
      id: socketId,
      emit: vi.fn(),
    };
    mockIO.sockets.sockets.get = (id) => id === socketId ? mockSocket : null;

    const response = await request(app)
      .post('/api/links')
      .send({ 
        url: 'https://example.com/test-delivered',
        deviceId: 'android-device-123'
      });

    expect(response.status).toBe(200);
    expect(response.body.success).toBe(true);
    expect(response.body.delivered).toBe(true);
    expect(response.body.message).toContain('delivered');

    // Verify socket emit was called
    expect(mockSocket.emit).toHaveBeenCalledWith(
      'link-received',
      expect.objectContaining({ url: 'https://example.com/test-delivered' })
    );
  });
});

describe('GET /api/links', () => {
  let app;

  beforeEach(async () => {
    // Wait for MongoDB to be ready
    const mongoose = await import('mongoose');
    if (mongoose.default.connection.readyState === 0) {
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    await cleanupTestData();
    app = express();
    app.use(express.json());
    app.use('/api/links', linksRouter);
  });

  afterEach(async () => {
    await cleanupTestData();
  });

  it('should return empty array when no links exist', async () => {
    const response = await request(app)
      .get('/api/links');

    expect(response.status).toBe(200);
    expect(response.body.links).toEqual([]);
  });

  it('should return links for user', async () => {
    const userId = 'test-user-123';
    const { Link } = await import('../../src/models/Link.js');

    // Create test links
    for (let i = 0; i < 5; i++) {
      await new Link({
        userId,
        deviceId: 'android-device',
        url: `https://example.com/test-${i}`,
        status: 'delivered',
      }).save();
    }

    const response = await request(app)
      .get('/api/links');

    expect(response.status).toBe(200);
    expect(response.body.links.length).toBe(5);
    expect(response.body.links[0].url).toContain('example.com');
  });
});

