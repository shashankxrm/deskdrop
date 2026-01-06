import express from 'express';
import { 
  generateRegistrationChallenge, 
  verifyRegistration,
  generateAuthenticationChallenge,
  verifyAuthentication 
} from '../services/webauthnService.js';
import { User } from '../models/User.js';
import { generateAccessToken, generateRefreshToken, verifyRefreshToken } from '../utils/jwt.js';

const router = express.Router();

// POST /api/auth/register/start - Begin registration
router.post('/register/start', async (req, res) => {
  try {
    const { email, userName } = req.body;
    
    if (!email || !userName) {
      return res.status(400).json({ 
        success: false,
        error: 'Email and userName are required' 
      });
    }
    
    // Validate email format
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      return res.status(400).json({ 
        success: false,
        error: 'Invalid email format' 
      });
    }
    
    // Check if user exists, get existing credentials
    let user = await User.findOne({ email });
    const existingCredentials = user?.credentials || [];
    
    // Generate userId if new user
    const userId = user?.userId || email; // Using email as userId for simplicity
    
    // Check if this is an Electron client (from User-Agent or custom header)
    const clientType = req.headers['x-client-type'] || 'web';
    
    // Generate registration challenge
    const options = await generateRegistrationChallenge(
      userId,
      userName,
      existingCredentials,
      clientType
    );
    
    res.json({ 
      success: true,
      options,
      challenge: options.challenge 
    });
  } catch (error) {
    console.error('Registration start error:', error);
    res.status(500).json({ 
      success: false,
      error: 'Failed to start registration',
      message: error.message 
    });
  }
});

// POST /api/auth/register/complete - Complete registration
router.post('/register/complete', async (req, res) => {
  try {
    const { email, userName, response, challenge, deviceName, deviceType, platform } = req.body;
    
    if (!email || !userName || !response || !challenge) {
      return res.status(400).json({ 
        success: false,
        error: 'Email, userName, response, and challenge are required' 
      });
    }
    
    // Find or create user
    let user = await User.findOne({ email });
    const userId = user?.userId || email;
    
    if (!user) {
      user = new User({
        userId,
        email,
        credentials: []
      });
    }
    
    // Verify registration
    const result = await verifyRegistration(
      userId, 
      response, 
      challenge,
      deviceName,
      deviceType,
      platform
    );
    
    if (result.verified) {
      // Generate JWT tokens
      const token = generateAccessToken(user.userId, user.email);
      const refreshToken = generateRefreshToken(user.userId);
      
      // Store refresh token in user document (optional, for revocation)
      user.refreshTokens.push({
        token: refreshToken,
        expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
        createdAt: new Date()
      });
      await user.save();
      
      res.json({ 
        success: true, 
        token,
        refreshToken,
        user: { 
          userId: user.userId, 
          email: user.email,
          credentialsCount: user.credentials.length
        }
      });
    } else {
      res.status(400).json({ 
        success: false,
        error: result.error || 'Registration verification failed' 
      });
    }
  } catch (error) {
    console.error('Registration complete error:', error);
    res.status(500).json({ 
      success: false,
      error: 'Registration failed',
      message: error.message 
    });
  }
});

// POST /api/auth/login/start - Begin authentication
router.post('/login/start', async (req, res) => {
  try {
    const { email } = req.body;
    
    if (!email) {
      return res.status(400).json({ 
        success: false,
        error: 'Email is required' 
      });
    }
    
    const user = await User.findOne({ email });
    if (!user || !user.credentials || user.credentials.length === 0) {
      return res.status(404).json({ 
        success: false,
        error: 'User not found or no credentials registered' 
      });
    }
    
    // Check if this is an Electron client
    const clientType = req.headers['x-client-type'] || 'web';
    
    // Generate authentication challenge
    const options = await generateAuthenticationChallenge(user.userId, clientType);
    
    res.json({ 
      success: true,
      options,
      challenge: options.challenge 
    });
  } catch (error) {
    console.error('Login start error:', error);
    res.status(500).json({ 
      success: false,
      error: 'Failed to start login',
      message: error.message 
    });
  }
});

// POST /api/auth/login/complete - Complete authentication
router.post('/login/complete', async (req, res) => {
  try {
    const { email, response, challenge } = req.body;
    
    if (!email || !response || !challenge) {
      return res.status(400).json({ 
        success: false,
        error: 'Email, response, and challenge are required' 
      });
    }
    
    const user = await User.findOne({ email });
    if (!user) {
      return res.status(404).json({ 
        success: false,
        error: 'User not found' 
      });
    }
    
    // Verify authentication
    const result = await verifyAuthentication(user.userId, response, challenge);
    
    if (result.verified) {
      // Generate JWT tokens
      const token = generateAccessToken(user.userId, user.email);
      const refreshToken = generateRefreshToken(user.userId);
      
      // Store refresh token
      user.refreshTokens.push({
        token: refreshToken,
        expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
        createdAt: new Date()
      });
      await user.save();
      
      res.json({ 
        success: true, 
        token,
        refreshToken,
        user: { 
          userId: user.userId, 
          email: user.email,
          credentialsCount: user.credentials.length
        }
      });
    } else {
      res.status(401).json({ 
        success: false,
        error: result.error || 'Authentication failed' 
      });
    }
  } catch (error) {
    console.error('Login complete error:', error);
    res.status(500).json({ 
      success: false,
      error: 'Authentication failed',
      message: error.message 
    });
  }
});

// POST /api/auth/logout - Logout (invalidate refresh token)
router.post('/logout', async (req, res) => {
  try {
    const { refreshToken } = req.body;
    
    if (refreshToken) {
      // Find and remove refresh token from user
      const decoded = verifyRefreshToken(refreshToken);
      const user = await User.findOne({ userId: decoded.userId });
      
      if (user) {
        user.refreshTokens = user.refreshTokens.filter(
          rt => rt.token !== refreshToken
        );
        await user.save();
      }
    }
    
    res.json({ 
      success: true, 
      message: 'Logged out successfully' 
    });
  } catch (error) {
    console.error('Logout error:', error);
    res.status(500).json({ 
      success: false,
      error: 'Logout failed' 
    });
  }
});

// POST /api/auth/refresh - Refresh access token
router.post('/refresh', async (req, res) => {
  try {
    const { refreshToken } = req.body;
    
    if (!refreshToken) {
      return res.status(400).json({ 
        success: false,
        error: 'Refresh token is required' 
      });
    }
    
    // Verify refresh token
    const decoded = verifyRefreshToken(refreshToken);
    
    // Check if token exists in user's refresh tokens
    const user = await User.findOne({ userId: decoded.userId });
    if (!user) {
      return res.status(404).json({ 
        success: false,
        error: 'User not found' 
      });
    }
    
    const tokenExists = user.refreshTokens.some(rt => rt.token === refreshToken);
    if (!tokenExists) {
      return res.status(401).json({ 
        success: false,
        error: 'Invalid refresh token' 
      });
    }
    
    // Generate new access token
    const token = generateAccessToken(user.userId, user.email);
    
    res.json({ 
      success: true,
      token 
    });
  } catch (error) {
    console.error('Refresh token error:', error);
    res.status(401).json({ 
      success: false,
      error: 'Invalid or expired refresh token' 
    });
  }
});

export default router;

