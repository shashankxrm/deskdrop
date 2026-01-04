import { io } from 'socket.io-client';

let socket = null;
let config = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;

export const setupSocketService = (callbacks) => {
  config = {
    serverUrl: process.env.BACKEND_URL || 'http://localhost:3000',
    pairingToken: process.env.PAIRING_TOKEN || null,
    onLinkReceived: callbacks?.onLinkReceived || null,
    onConnectionStatusChange: callbacks?.onConnectionStatusChange || null
  };

  // Try to load pairing token from config file or environment
  // Will be called after Electron app is ready
  loadPairingTokenFromConfig();

  if (!config.pairingToken) {
    console.error('Pairing token not found. Please set PAIRING_TOKEN environment variable or create config file.');
    if (config.onConnectionStatusChange) {
      config.onConnectionStatusChange('disconnected', 'No pairing token');
    }
    return {
      connect: () => {},
      disconnect: () => {},
      reconnect: () => {},
      isConnected: () => false
    };
  }

  connect();

  return {
    connect,
    disconnect,
    reconnect,
    isConnected: () => socket?.connected || false
  };
};

async function loadPairingTokenFromConfig() {
  // Try to load from environment variable first
  if (process.env.PAIRING_TOKEN) {
    config.pairingToken = process.env.PAIRING_TOKEN;
    return;
  }

  // Try to load from config file (for production use)
  try {
    const fs = await import('fs');
    const path = await import('path');
    const electron = await import('electron');
    const userDataPath = electron.app.getPath('userData');
    const configPath = path.join(userDataPath, 'config.json');
    
    if (fs.existsSync(configPath)) {
      const configData = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      if (configData.pairingToken) {
        config.pairingToken = configData.pairingToken;
        config.serverUrl = configData.serverUrl || config.serverUrl;
      }
    }
  } catch (error) {
    // Config file not available or Electron not ready - will use env var or fail gracefully
    console.log('Could not load config file (this is OK if using env vars):', error.message);
  }
}

function connect() {
  if (socket?.connected) {
    return;
  }

  console.log(`Connecting to ${config.serverUrl}...`);

  socket = io(config.serverUrl, {
    auth: {
      pairingToken: config.pairingToken
    },
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: MAX_RECONNECT_ATTEMPTS
  });

  socket.on('connect', () => {
    console.log('Connected to backend');
    reconnectAttempts = 0;
    if (config.onConnectionStatusChange) {
      config.onConnectionStatusChange('connected', 'Connected');
    }
  });

  socket.on('disconnect', (reason) => {
    console.log('Disconnected from backend:', reason);
    if (config.onConnectionStatusChange) {
      config.onConnectionStatusChange('disconnected', reason);
    }
  });

  socket.on('connect_error', (error) => {
    console.error('Connection error:', error.message);
    reconnectAttempts++;
    
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      console.error('Max reconnection attempts reached');
      if (config.onConnectionStatusChange) {
        config.onConnectionStatusChange('error', 'Max reconnection attempts reached');
      }
    } else {
      if (config.onConnectionStatusChange) {
        config.onConnectionStatusChange('connecting', `Reconnecting... (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);
      }
    }
  });

  socket.on('link-received', (linkData) => {
    console.log('Link received via Socket.IO:', linkData);
    if (config.onLinkReceived) {
      config.onLinkReceived(linkData);
    }
  });

  socket.on('pong', () => {
    // Keepalive response
  });

  // Send ping periodically
  setInterval(() => {
    if (socket?.connected) {
      socket.emit('ping');
    }
  }, 30000); // Every 30 seconds
}

function disconnect() {
  if (socket) {
    socket.disconnect();
    socket = null;
  }
}

function reconnect() {
  disconnect();
  setTimeout(() => {
    connect();
  }, 1000);
}

