import { io } from 'socket.io-client';

let socket = null;
let config = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;

export const setupSocketService = async (callbacks) => {
  config = {
    serverUrl: process.env.BACKEND_URL || 'http://localhost:3000',
    pairingToken: process.env.PAIRING_TOKEN || null,
    onLinkReceived: callbacks?.onLinkReceived || null,
    onConnectionStatusChange: callbacks?.onConnectionStatusChange || null
  };

  // Try to load pairing token from config file or environment
  // Wait for Electron app to be ready
  await loadPairingTokenFromConfig();
  
  console.log('Final config - serverUrl:', config.serverUrl);
  console.log('Final config - pairingToken:', config.pairingToken ? '***' + config.pairingToken.slice(-4) : 'NOT SET');

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
  // Priority: Config file first (for production), then environment variables (for development)
  
  // Try to load from config file first (production use)
  try {
    const { readFileSync, existsSync } = await import('fs');
    const { join } = await import('path');
    const { app } = await import('electron');
    
    // Wait a bit for app to be ready
    if (!app || !app.isReady()) {
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
    
    const userDataPath = app.getPath('userData');
    const configPath = join(userDataPath, 'config.json');
    
    if (existsSync(configPath)) {
      const configData = JSON.parse(readFileSync(configPath, 'utf8'));
      console.log('Loaded config from file:', configPath);
      
      if (configData.pairingToken) {
        config.pairingToken = configData.pairingToken;
        console.log('Using pairing token from config file');
      }
      if (configData.serverUrl) {
        config.serverUrl = configData.serverUrl;
        console.log('Using server URL from config file:', config.serverUrl);
        return; // Config file takes priority, don't check env vars
      }
    }
  } catch (error) {
    console.log('Could not load config file, falling back to environment variables:', error.message);
  }
  
  // Fall back to environment variables (for development)
  if (process.env.PAIRING_TOKEN) {
    config.pairingToken = process.env.PAIRING_TOKEN;
    console.log('Using PAIRING_TOKEN from environment');
  }
  if (process.env.BACKEND_URL) {
    config.serverUrl = process.env.BACKEND_URL;
    console.log('Using BACKEND_URL from environment:', config.serverUrl);
  }
}

function connect() {
  if (socket?.connected) {
    return;
  }

  console.log(`Connecting to ${config.serverUrl}...`);
  console.log('Using pairing token:', config.pairingToken ? '***' + config.pairingToken.slice(-4) : 'NOT SET');

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

