import express from 'express';
import { deliverLink } from '../services/linkService.js';
import { authenticate } from '../middleware/auth.js';

const router = express.Router();

// POST /api/links - Receive link from Android
router.post('/', authenticate, async (req, res) => {
  try {
    const { url } = req.body;
    const userId = req.userId; // Set by auth middleware
    const deviceId = req.deviceId || req.body.deviceId; // Can be passed in body or set by auth

    if (!url) {
      return res.status(400).json({ error: 'URL is required' });
    }

    // Validate URL format
    try {
      new URL(url);
    } catch (e) {
      return res.status(400).json({ error: 'Invalid URL format' });
    }

    if (!userId) {
      return res.status(401).json({ error: 'Authentication required' });
    }

    if (!deviceId) {
      return res.status(400).json({ error: 'Device ID is required' });
    }

    // Deliver link (will queue if device offline)
    const result = await deliverLink(userId, deviceId, url, req.app.get('io'));

    if (result.success) {
      res.status(200).json({
        success: true,
        delivered: result.delivered,
        queued: result.queued,
        message: result.delivered ? 'Link delivered' : 'Link queued for delivery'
      });
    } else {
      res.status(500).json({ error: result.error || 'Failed to process link' });
    }
  } catch (error) {
    console.error('Error in POST /api/links:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// GET /api/links - Get link history (optional for MVP)
router.get('/', authenticate, async (req, res) => {
  try {
    const userId = req.userId;
    if (!userId) {
      return res.status(401).json({ error: 'Authentication required' });
    }

    const { Link } = await import('../models/Link.js');
    const links = await Link.find({ userId })
      .sort({ createdAt: -1 })
      .limit(50)
      .select('url status createdAt deliveredAt');

    res.json({ links });
  } catch (error) {
    console.error('Error in GET /api/links:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;

