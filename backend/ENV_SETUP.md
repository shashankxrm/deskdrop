# Environment Variables Setup Guide

This guide explains what to configure in your `.env` file based on your setup.

## Quick Start (Local Development)

If you're running MongoDB and Redis locally, you can use these defaults:

```env
PORT=3000
NODE_ENV=development

MONGODB_URI=mongodb://localhost:27017/deskdrop

REDIS_HOST=localhost
REDIS_PORT=6379

DEV_TOKEN=dev-token-for-testing-change-in-production
```

**No changes needed** - just copy `.env.example` to `.env` and you're good to go!

---

## Detailed Configuration

### 1. Server Configuration

```env
PORT=3000
NODE_ENV=development
```

- **PORT**: Change only if port 3000 is already in use (e.g., `3001`, `8080`)
- **NODE_ENV**: Keep as `development` for local dev, use `production` for deployment

### 2. MongoDB Configuration

#### Option A: Local MongoDB (Default)
```env
MONGODB_URI=mongodb://localhost:27017/deskdrop
```
- Works if you have MongoDB installed locally
- Database name: `deskdrop` (will be created automatically)

#### Option B: MongoDB Atlas (Cloud)
```env
MONGODB_URI=mongodb+srv://username:password@cluster.mongodb.net/deskdrop
```
- Replace `username` and `password` with your Atlas credentials
- Replace `cluster.mongodb.net` with your cluster URL
- Get connection string from MongoDB Atlas dashboard

**To get MongoDB Atlas connection string:**
1. Go to https://cloud.mongodb.com
2. Create a free cluster
3. Click "Connect" → "Connect your application"
4. Copy the connection string
5. Replace `<password>` with your database password

### 3. Redis Configuration

#### Option A: Local Redis (Default)
```env
REDIS_HOST=localhost
REDIS_PORT=6379
```
- Works if you have Redis installed locally
- Install Redis: `brew install redis` (macOS) or `sudo apt-get install redis` (Linux)

#### Option B: Redis Cloud
```env
REDIS_URL=redis://username:password@host:port
```
- Get connection details from your Redis cloud provider
- Comment out `REDIS_HOST` and `REDIS_PORT` if using `REDIS_URL`

**To get Redis Cloud connection:**
1. Sign up at https://redis.com/try-free/ or https://upstash.com
2. Create a database
3. Copy the connection URL/credentials

### 4. Authentication

```env
DEV_TOKEN=dev-token-for-testing-change-in-production
```

- **For MVP testing**: Keep the default or change to any string you want
- This token is used by Android app and API calls
- **Important**: Change this in production (not implemented yet for MVP)

**Example:**
```env
DEV_TOKEN=my-secret-dev-token-12345
```

### 5. WebAuthn Configuration

```env
WEBAUTHN_RP_ID=localhost
WEBAUTHN_RP_NAME=DeskDrop
WEBAUTHN_ORIGIN=http://localhost:3000
```

- **WEBAUTHN_RP_ID**: Relying Party ID (domain name)
  - For local dev: `localhost`
  - For production: Your domain (e.g., `deskdrop-backend.onrender.com`)
- **WEBAUTHN_RP_NAME**: Display name for your app
- **WEBAUTHN_ORIGIN**: Full origin URL
  - For local dev: `http://localhost:3000`
  - For production: `https://your-domain.com`

**Important Notes:**
- `RP_ID` must match your domain (no protocol, no port for HTTPS)
- `ORIGIN` must include protocol and port
- For production, ensure your domain matches the RP_ID

### 6. JWT Configuration

```env
JWT_SECRET=your-jwt-secret-key-here
JWT_REFRESH_SECRET=your-refresh-secret-key-here
JWT_EXPIRES_IN=7d
JWT_REFRESH_EXPIRES_IN=30d
```

- **JWT_SECRET**: Secret key for signing access tokens (use a strong random string)
- **JWT_REFRESH_SECRET**: Secret key for signing refresh tokens (different from JWT_SECRET)
- **JWT_EXPIRES_IN**: Access token expiration (default: 7 days)
- **JWT_REFRESH_EXPIRES_IN**: Refresh token expiration (default: 30 days)

**Security Notes:**
- Generate strong random secrets for production
- Never commit secrets to version control
- Use different secrets for development and production
- Example: `openssl rand -base64 32` to generate secrets

---

## Common Setup Scenarios

### Scenario 1: Everything Local (Easiest)
```env
PORT=3000
NODE_ENV=development
MONGODB_URI=mongodb://localhost:27017/deskdrop
REDIS_HOST=localhost
REDIS_PORT=6379
DEV_TOKEN=dev-token-for-testing-change-in-production
```

**Prerequisites:**
- MongoDB installed and running locally
- Redis installed and running locally

### Scenario 2: MongoDB Atlas + Local Redis
```env
PORT=3000
NODE_ENV=development
MONGODB_URI=mongodb+srv://myuser:mypassword@cluster0.xxxxx.mongodb.net/deskdrop
REDIS_HOST=localhost
REDIS_PORT=6379
DEV_TOKEN=dev-token-for-testing-change-in-production
```

### Scenario 3: Everything Cloud (MongoDB Atlas + Redis Cloud)
```env
PORT=3000
NODE_ENV=development
MONGODB_URI=mongodb+srv://myuser:mypassword@cluster0.xxxxx.mongodb.net/deskdrop
REDIS_URL=redis://default:password@redis-xxxxx.upstash.io:6379
DEV_TOKEN=dev-token-for-testing-change-in-production
```

---

## Verification

After setting up your `.env` file, test the connection:

1. **Start MongoDB** (if local):
   ```bash
   mongod
   # or
   brew services start mongodb-community
   ```

2. **Start Redis** (if local):
   ```bash
   redis-server
   # or
   brew services start redis
   ```

3. **Start the backend**:
   ```bash
   cd backend
   npm install
   npm start
   ```

4. **Test health endpoint**:
   ```bash
   curl http://localhost:3000/api/health
   ```

If you see `{"status":"ok",...}`, your configuration is working!

---

## Troubleshooting

### MongoDB Connection Error
- Check if MongoDB is running: `mongosh` or `mongo`
- Verify connection string format
- Check firewall/network settings for Atlas

### Redis Connection Error
- Check if Redis is running: `redis-cli ping` (should return `PONG`)
- Verify Redis host/port
- For cloud Redis, check credentials and network access

### Port Already in Use
- Change `PORT=3000` to another port (e.g., `3001`)
- Update Android app's `ApiClient.kt` with new port
- Update desktop app's `BACKEND_URL` environment variable

