import { Link } from '../models/Link.js';
import { Device } from '../models/Device.js';
import { queueLinkForDevice, getQueuedLinksForDevice, clearQueueForDevice } from './redisService.js';

export const deliverLink = async (userId, deviceId, url, io) => {
  try {
    // Store link in MongoDB (deviceId here is the Android device that sent it)
    const link = new Link({
      url,
      userId,
      deviceId, // Android device ID
      status: 'pending'
    });
    await link.save();
    console.log(`Link saved to database: ${url} from Android device ${deviceId}`);

    // Find the desktop device for this user (desktop devices have pairingToken)
    const desktopDevice = await Device.findOne({ 
      userId,
      pairingToken: { $ne: null } // Desktop devices have pairing tokens
    });

    if (!desktopDevice) {
      console.log(`No desktop device found for user ${userId}`);
      // Queue for when desktop device connects
      await queueLinkForDevice(`user-${userId}`, {
        url,
        linkId: link._id.toString(),
        userId,
        timestamp: link.createdAt
      });
      return { success: true, delivered: false, queued: true, message: 'No desktop device connected' };
    }

    // If desktop device is online and has socketId, deliver immediately
    if (desktopDevice.isOnline && desktopDevice.socketId) {
      const socket = io.sockets.sockets.get(desktopDevice.socketId);
      if (socket) {
        socket.emit('link-received', {
          url,
          linkId: link._id.toString(),
          timestamp: link.createdAt
        });
        link.status = 'delivered';
        link.deliveredAt = new Date();
        await link.save();
        console.log(`Link delivered to desktop device ${desktopDevice.deviceId} via Socket.IO`);
        return { success: true, delivered: true };
      }
    }

    // Desktop device is offline, queue in Redis
    await queueLinkForDevice(desktopDevice.deviceId, {
      url,
      linkId: link._id.toString(),
      userId,
      timestamp: link.createdAt
    });
    console.log(`Link queued for offline desktop device ${desktopDevice.deviceId}`);
    return { success: true, delivered: false, queued: true };
  } catch (error) {
    console.error('Error delivering link:', error);
    return { success: false, error: error.message };
  }
};

export const processQueuedLinksForDevice = async (deviceId, socketId, io) => {
  try {
    // Get device to find userId
    const device = await Device.findOne({ deviceId });
    if (!device) {
      console.log(`Device ${deviceId} not found for processing queued links`);
      return;
    }

    const userId = device.userId;
    
    // Check both queue keys: device-specific and user-specific (for links queued before device was registered)
    const deviceQueuedLinks = await getQueuedLinksForDevice(deviceId);
    const userQueuedLinks = await getQueuedLinksForDevice(`user-${userId}`);
    
    const allQueuedLinks = [...deviceQueuedLinks, ...userQueuedLinks];
    
    if (allQueuedLinks.length === 0) {
      console.log(`No queued links found for device ${deviceId} or user ${userId}`);
      return;
    }

    console.log(`Processing ${allQueuedLinks.length} queued links for device ${deviceId} (user: ${userId})`);
    
    const socket = io.sockets.sockets.get(socketId);
    if (!socket) {
      console.log(`Socket ${socketId} not found`);
      return;
    }

    // Deliver all queued links
    for (const linkData of allQueuedLinks) {
      socket.emit('link-received', linkData);
      
      // Update link status in database
      const link = await Link.findById(linkData.linkId);
      if (link) {
        link.status = 'delivered';
        link.deliveredAt = new Date();
        await link.save();
      }
    }

    // Clear both queues
    await clearQueueForDevice(deviceId);
    await clearQueueForDevice(`user-${userId}`);
    console.log(`Delivered and cleared ${allQueuedLinks.length} queued links`);
  } catch (error) {
    console.error('Error processing queued links:', error);
  }
};

