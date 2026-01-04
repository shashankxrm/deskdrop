import express from 'express';
import { Device } from '../models/Device.js';
import crypto from 'crypto';
import { authenticateDevToken } from '../middleware/auth.js';

const router = express.Router();

// POST /api/devices/pair - Device pairing endpoint
router.post('/pair', authenticateDevToken, async (req, res) => {
  try {
    const userId = req.userId;
    const { pairingToken, deviceName } = req.body;

    if (!pairingToken) {
      return res.status(400).json({ error: 'Pairing token is required' });
    }

    // Find device by pairing token
    const device = await Device.findOne({ pairingToken });
    
    if (!device) {
      return res.status(404).json({ error: 'Invalid pairing token' });
    }

    // Associate device with user
    device.userId = userId;
    if (deviceName) {
      device.deviceName = deviceName;
    }
    await device.save();

    res.json({
      success: true,
      deviceId: device.deviceId,
      deviceName: device.deviceName,
      message: 'Device paired successfully'
    });
  } catch (error) {
    console.error('Error in POST /api/devices/pair:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// POST /api/devices/generate-pairing-token - Generate new pairing token for desktop
router.post('/generate-pairing-token', authenticateDevToken, async (req, res) => {
  try {
    const userId = req.userId;
    const { deviceName } = req.body;

    // Generate unique device ID and pairing token
    const deviceId = `device-${crypto.randomBytes(16).toString('hex')}`;
    const pairingToken = crypto.randomBytes(32).toString('hex');

    // Create device record
    const device = new Device({
      deviceId,
      userId,
      pairingToken,
      deviceName: deviceName || 'Desktop Device',
      isOnline: false
    });

    await device.save();

    res.json({
      success: true,
      deviceId,
      pairingToken,
      message: 'Pairing token generated successfully'
    });
  } catch (error) {
    console.error('Error in POST /api/devices/generate-pairing-token:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// GET /api/devices - Get user's devices
router.get('/', authenticateDevToken, async (req, res) => {
  try {
    const userId = req.userId;
    const devices = await Device.find({ userId })
      .select('deviceId deviceName isOnline lastSeen createdAt');

    res.json({ devices });
  } catch (error) {
    console.error('Error in GET /api/devices:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;

