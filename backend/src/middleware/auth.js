// Dev token authentication middleware (for MVP testing)
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

// WebAuthn authentication middleware (placeholder for future implementation)
export const authenticateWebAuthn = async (req, res, next) => {
  // TODO: Implement WebAuthn verification
  // For now, fall back to dev token
  return authenticateDevToken(req, res, next);
};

