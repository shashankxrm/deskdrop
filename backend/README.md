# DeskDrop Backend

Node.js backend server for DeskDrop MVP - handles link ingestion from Android and real-time delivery to desktop clients.

## Setup

1. Install dependencies:
```bash
npm install
```

2. Copy `.env.example` to `.env` and configure:
```bash
cp .env.example .env
```

3. Update `.env` with your MongoDB and Redis connection strings.

4. Start the server:
```bash
npm start
```

For development with auto-reload:
```bash
npm run dev
```

## Environment Variables

- `PORT` - Server port (default: 3000)
- `MONGODB_URI` - MongoDB connection string
- `REDIS_HOST` - Redis host (default: localhost)
- `REDIS_PORT` - Redis port (default: 6379)
- `REDIS_URL` - Alternative: full Redis URL
- `DEV_TOKEN` - Development token for testing (change in production)

## API Endpoints

### Health Check
- `GET /api/health` - Server health status

### Links
- `POST /api/links` - Submit a link from Android
  - Headers: `Authorization: Bearer <dev-token>`
  - Body: `{ "url": "https://...", "deviceId": "device-123" }`
- `GET /api/links` - Get link history (requires auth)

### Authentication
- `POST /api/auth/register` - User registration (WebAuthn - placeholder)
- `POST /api/auth/login` - User login (WebAuthn - placeholder)

### Devices
- `POST /api/devices/pair` - Pair a desktop device with pairing token
- `POST /api/devices/generate-pairing-token` - Generate new pairing token
- `GET /api/devices` - Get user's devices

## Socket.IO

Desktop clients connect via Socket.IO with authentication:
- Connection requires `pairingToken` in handshake auth
- Events:
  - `link-received` - Desktop receives link data
  - `ping` / `pong` - Keepalive

## Development Token

For MVP testing, use the `DEV_TOKEN` from `.env`:
```
Authorization: Bearer <DEV_TOKEN>
```

The dev token allows bypassing WebAuthn for early development.

