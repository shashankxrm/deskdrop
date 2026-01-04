# DeskDrop Desktop App

Electron desktop application that receives links from Android via Socket.IO and copies them to the clipboard.

## Setup

1. Install dependencies:
```bash
npm install
```

2. Set environment variables or create config file:
   - `BACKEND_URL` - Backend server URL (default: http://localhost:3000)
   - `PAIRING_TOKEN` - Device pairing token from backend

3. Start the app:
```bash
npm start
```

## Configuration

### Environment Variables
```bash
export BACKEND_URL=http://localhost:3000
export PAIRING_TOKEN=your-pairing-token-here
npm start
```

### Config File (Alternative)
The app will look for a config file at `~/Library/Application Support/deskdrop-desktop/config.json` (macOS) or equivalent on other platforms:

```json
{
  "serverUrl": "http://localhost:3000",
  "pairingToken": "your-pairing-token-here"
}
```

### Getting a Pairing Token

1. Start the backend server
2. Use the API to generate a pairing token:
```bash
curl -X POST http://localhost:3000/api/devices/generate-pairing-token \
  -H "Authorization: Bearer <DEV_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"deviceName": "My Desktop"}'
```

3. Use the returned `pairingToken` in the desktop app

## Features

- **System Tray**: Runs in background, accessible via system tray icon
- **Real-time Delivery**: Receives links via Socket.IO from backend
- **Clipboard Integration**: Automatically copies received links to clipboard
- **Auto-open Browser**: Optionally opens links in default browser
- **Connection Status**: Shows connection status in tray menu

## Usage

1. Start the desktop app (runs in background)
2. App connects to backend using pairing token
3. When Android sends a link, it's automatically:
   - Copied to clipboard
   - Opened in default browser (optional)
   - Shown in notification (if supported)

## Building

To package the app for distribution:
```bash
npm install -g electron-builder
electron-builder
```

