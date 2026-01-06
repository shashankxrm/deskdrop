import { verifyAccessToken } from '../utils/jwt.js';

// Dev token authentication middleware (for MVP testing and backward compatibility)
export const authenticateDevToken = (req, res, next) => {
  const authHeader = req.headers.authorization;
  const devToken = process.env.DEV_TOKEN || 'dev-token-for-testing-change-in-production';

  if (authHeader && authHeader.startsWith('Bearer ')) {
    const token = authHeader.substring(7);
    if (token === devToken) {
      // For dev token, use a default userId or extract from token
      // For MVP, we'll use a simple default user
      req.userId = req.body.userId || 'dev-user-1';
      req.deviceId = req.body.deviceId;
      return next();
    }
  }

  // Also check for token in query params (for easier testing)
  const queryToken = req.query.token;
  if (queryToken === devToken) {
    req.userId = req.body.userId || req.query.userId || 'dev-user-1';
    req.deviceId = req.body.deviceId || req.query.deviceId;
    return next();
  }

  res.status(401).json({ error: 'Invalid or missing authentication token' });
};

// WebAuthn JWT authentication middleware
export const authenticateWebAuthn = (req, res, next) => {
  const authHeader = req.headers.authorization;
  
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    // Fall back to dev token for backward compatibility
    return authenticateDevToken(req, res, next);
  }
  
  const token = authHeader.substring(7);
  
  try {
    const decoded = verifyAccessToken(token);
    req.userId = decoded.userId;
    req.email = decoded.email;
    next();
  } catch (error) {
    // If JWT verification fails, try dev token as fallback
    if (token === (process.env.DEV_TOKEN || 'dev-token-for-testing-change-in-production')) {
      return authenticateDevToken(req, res, next);
    }
    res.status(401).json({ 
      success: false,
      error: 'Invalid or expired token',
      message: error.message 
    });
  }
};

// Combined authentication: tries WebAuthn JWT first, falls back to dev token
export const authenticate = authenticateWebAuthn;

