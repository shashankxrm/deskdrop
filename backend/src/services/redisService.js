import { createClient } from 'redis';

let redisClient = null;

export const connectRedis = async () => {
  try {
    const redisUrl = process.env.REDIS_URL || `redis://${process.env.REDIS_HOST || 'localhost'}:${process.env.REDIS_PORT || 6379}`;
    
    redisClient = createClient({
      url: redisUrl
    });

    redisClient.on('error', (err) => {
      console.error('Redis Client Error:', err);
    });

    await redisClient.connect();
    console.log('Redis connected successfully');
    return redisClient;
  } catch (error) {
    console.error('Failed to connect to Redis:', error);
    throw error;
  }
};

export const getRedisClient = () => {
  if (!redisClient) {
    throw new Error('Redis client not initialized. Call connectRedis() first.');
  }
  return redisClient;
};

export const queueLinkForDevice = async (deviceId, linkData) => {
  const client = getRedisClient();
  const queueKey = `queue:device:${deviceId}`;
  await client.lPush(queueKey, JSON.stringify(linkData));
  console.log(`Queued link for device ${deviceId}`);
};

export const getQueuedLinksForDevice = async (deviceId) => {
  const client = getRedisClient();
  const queueKey = `queue:device:${deviceId}`;
  const links = await client.lRange(queueKey, 0, -1);
  return links.map(link => JSON.parse(link));
};

export const clearQueueForDevice = async (deviceId) => {
  const client = getRedisClient();
  const queueKey = `queue:device:${deviceId}`;
  await client.del(queueKey);
  console.log(`Cleared queue for device ${deviceId}`);
};

