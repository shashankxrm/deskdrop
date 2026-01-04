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
    // Get all online desktop devices for this user
    const onlineDesktopDevices = await Device.find({ 
      userId,
      pairingToken: { $ne: null },
      isOnline: true,
      socketId: { $ne: null }
    });

    // Find the device with a valid socket connection
    let desktopDevice = null;
    for (const device of onlineDesktopDevices) {
      const socket = io.sockets.sockets.get(device.socketId);
      if (socket) {
        desktopDevice = device;
        console.log(`Found valid online desktop device: ${device.deviceId}`);
        break;
      } else {
        // Socket doesn't exist, mark device as offline
        console.log(`Socket ${device.socketId} for device ${device.deviceId} doesn't exist, marking offline`);
        device.isOnline = false;
        device.socketId = null;
        await device.save();
      }
    }

    // If no valid online device, find any desktop device for this user (offline)
    if (!desktopDevice) {
      desktopDevice = await Device.findOne({ 
        userId,
        pairingToken: { $ne: null }
      });
    }

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
      } else {
        // Socket not found, mark device as offline
        console.log(`Socket ${desktopDevice.socketId} not found, marking device as offline`);
        desktopDevice.isOnline = false;
        desktopDevice.socketId = null;
        await desktopDevice.save();
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
    
    // Find all desktop devices for this user (to check their queues)
    const allUserDevices = await Device.find({ 
      userId,
      pairingToken: { $ne: null } // All desktop devices for this user
    });
    
    // Check queues for: current device, user queue, and all other devices for this user
    const deviceQueuedLinks = await getQueuedLinksForDevice(deviceId);
    const userQueuedLinks = await getQueuedLinksForDevice(`user-${userId}`);
    
    // Also check queues for other devices (in case links were queued for old devices)
    const otherDeviceQueues = await Promise.all(
      allUserDevices
        .filter(d => d.deviceId !== deviceId)
        .map(d => getQueuedLinksForDevice(d.deviceId))
    );
    const otherDeviceLinks = otherDeviceQueues.flat();
    
    const allQueuedLinks = [...deviceQueuedLinks, ...userQueuedLinks, ...otherDeviceLinks];
    
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

    // Clear all queues: current device, user queue, and all other devices for this user
    await clearQueueForDevice(deviceId);
    await clearQueueForDevice(`user-${userId}`);
    for (const otherDevice of allUserDevices) {
      if (otherDevice.deviceId !== deviceId) {
        await clearQueueForDevice(otherDevice.deviceId);
      }
    }
    console.log(`Delivered and cleared ${allQueuedLinks.length} queued links`);
  } catch (error) {
    console.error('Error processing queued links:', error);
  }
};

