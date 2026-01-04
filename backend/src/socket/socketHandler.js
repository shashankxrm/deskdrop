import { Device } from '../models/Device.js';
import { processQueuedLinksForDevice } from '../services/linkService.js';

export const setupSocketIO = (io) => {
  io.use(async (socket, next) => {
    // Authenticate socket connection using pairing token
    const pairingToken = socket.handshake.auth.pairingToken;
    
    if (!pairingToken) {
      return next(new Error('Pairing token required'));
    }

    try {
      const device = await Device.findOne({ pairingToken });
      if (!device) {
        return next(new Error('Invalid pairing token'));
      }

      // Attach device info to socket
      socket.deviceId = device.deviceId;
      socket.userId = device.userId;
      socket.pairingToken = pairingToken;

      next();
    } catch (error) {
      console.error('Socket authentication error:', error);
      next(new Error('Authentication failed'));
    }
  });

  io.on('connection', async (socket) => {
    console.log(`Device connected: ${socket.deviceId} (User: ${socket.userId})`);

    // Update device status
    try {
      const device = await Device.findOne({ deviceId: socket.deviceId });
      if (device) {
        device.isOnline = true;
        device.socketId = socket.id;
        device.lastSeen = new Date();
        await device.save();

        // Process any queued links for this device
        await processQueuedLinksForDevice(socket.deviceId, socket.id, io);
      }
    } catch (error) {
      console.error('Error updating device status:', error);
    }

    // Handle disconnection
    socket.on('disconnect', async () => {
      console.log(`Device disconnected: ${socket.deviceId}`);
      
      try {
        const device = await Device.findOne({ deviceId: socket.deviceId });
        if (device) {
          device.isOnline = false;
          device.socketId = null;
          await device.save();
        }
      } catch (error) {
        console.error('Error updating device status on disconnect:', error);
      }
    });

    // Handle ping/pong for keepalive
    socket.on('ping', () => {
      socket.emit('pong');
    });
  });

  return io;
};

