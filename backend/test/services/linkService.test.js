import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { deliverLink, processQueuedLinksForDevice } from '../../src/services/linkService.js';
import { createTestDevice, cleanupTestData, createMockSocketIO } from '../helpers.js';
import * as redisService from '../../src/services/redisService.js';

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

describe('linkService', () => {
  beforeEach(async () => {
    // Wait for MongoDB to be ready
    const mongoose = await import('mongoose');
    if (mongoose.default.connection.readyState === 0) {
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    await cleanupTestData();
    mockRedisStore.clear();
    vi.clearAllMocks();
  });

  afterEach(async () => {
    await cleanupTestData();
    mockRedisStore.clear();
  });

  describe('deliverLink', () => {
    // Note: These tests verify error handling when invalid data is passed
    // The service catches Mongoose validation errors and returns them
    it('should return error when userId is missing', async () => {
      const { mockIO } = createMockSocketIO();
      
      const result = await deliverLink(null, 'device-1', 'https://example.com', mockIO);
      expect(result.success).toBe(false);
      expect(result.error).toBeTruthy();
    });

    it('should return error when deviceId is missing', async () => {
      const { mockIO } = createMockSocketIO();
      
      const result = await deliverLink('user-1', null, 'https://example.com', mockIO);
      expect(result.success).toBe(false);
      expect(result.error).toBeTruthy();
    });

    it('should save link to database', async () => {
      const { mockIO } = createMockSocketIO();
      const userId = 'user-1';
      const deviceId = 'android-device-1';
      const url = 'https://example.com/test';

      const result = await deliverLink(userId, deviceId, url, mockIO);

      expect(result.success).toBe(true);
      expect(result.queued).toBe(true); // No desktop device, so queued

      // Verify link was saved
      const { Link } = await import('../../src/models/Link.js');
      const savedLink = await Link.findOne({ url, userId, deviceId });
      expect(savedLink).toBeTruthy();
      expect(savedLink.status).toBe('pending');
    });

    it('should deliver link to online desktop device', async () => {
      const userId = 'user-1';
      const socketId = 'socket-123';
      const desktopDevice = await createTestDevice({
        userId,
        deviceType: 'desktop',
        isOnline: true,
        socketId,
        pairingToken: 'pairing-token-1',
      });
      
      const { mockIO, mockSocket } = createMockSocketIO();
      mockIO.addSocket(socketId, mockSocket);

      const url = 'https://example.com/test';
      const result = await deliverLink(userId, 'android-device-1', url, mockIO);

      expect(result.success).toBe(true);
      expect(result.delivered).toBe(true);

      // Verify link status updated
      const { Link } = await import('../../src/models/Link.js');
      const link = await Link.findOne({ url });
      expect(link.status).toBe('delivered');
      expect(link.deliveredAt).toBeTruthy();

      // Verify socket event was emitted
      const events = mockIO.getEmittedEvents();
      const linkEvent = events.find(e => e.event === 'link-received');
      expect(linkEvent).toBeTruthy();
      expect(linkEvent.data.url).toBe(url);
    });

    it('should queue link when desktop device is offline', async () => {
      const userId = 'user-1';
      const desktopDevice = await createTestDevice({
        userId,
        deviceType: 'desktop',
        isOnline: false,
        pairingToken: 'pairing-token-1',
      });

      const { mockIO } = createMockSocketIO();
      const url = 'https://example.com/test';
      const result = await deliverLink(userId, 'android-device-1', url, mockIO);

      expect(result.success).toBe(true);
      expect(result.delivered).toBe(false);
      expect(result.queued).toBe(true);

      // Verify link was queued in Redis
      expect(redisService.queueLinkForDevice).toHaveBeenCalledWith(
        desktopDevice.deviceId,
        expect.objectContaining({ url })
      );
    });
  });

  describe('processQueuedLinksForDevice', () => {
    it('should process queued links when device comes online', async () => {
      const userId = 'user-1';
      const socketId = 'socket-123';
      const device = await createTestDevice({
        userId,
        deviceType: 'desktop',
        pairingToken: 'pairing-token-1',
        socketId,
      });

      const { mockIO, mockSocket } = createMockSocketIO();
      mockIO.addSocket(socketId, mockSocket);

      // Queue some links
      const { Link } = await import('../../src/models/Link.js');
      const link1 = new Link({
        userId,
        deviceId: 'android-1',
        url: 'https://example.com/1',
        status: 'pending',
      });
      await link1.save();

      const link2 = new Link({
        userId,
        deviceId: 'android-2',
        url: 'https://example.com/2',
        status: 'pending',
      });
      await link2.save();

      await redisService.queueLinkForDevice(device.deviceId, {
        url: link1.url,
        linkId: link1._id.toString(),
        userId,
        timestamp: link1.createdAt,
      });

      await redisService.queueLinkForDevice(device.deviceId, {
        url: link2.url,
        linkId: link2._id.toString(),
        userId,
        timestamp: link2.createdAt,
      });

      // Process queued links
      await processQueuedLinksForDevice(device.deviceId, socketId, mockIO);

      // Verify links were delivered
      const events = mockIO.getEmittedEvents();
      const linkEvents = events.filter(e => e.event === 'link-received');
      expect(linkEvents.length).toBe(2);

      // Verify links status updated
      const updatedLink1 = await Link.findById(link1._id);
      const updatedLink2 = await Link.findById(link2._id);
      expect(updatedLink1.status).toBe('delivered');
      expect(updatedLink2.status).toBe('delivered');

      // Verify queue was cleared
      expect(redisService.clearQueueForDevice).toHaveBeenCalledWith(device.deviceId);
    });
  });
});

