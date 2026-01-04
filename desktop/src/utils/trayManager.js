import { app, Tray, Menu, nativeImage } from 'electron';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

let tray = null;
let connectionStatus = 'disconnected';
let reconnectCallback = null;

export const setupTrayManager = (callbacks) => {
  reconnectCallback = callbacks?.onReconnect || null;
  
  // Create tray icon (using a simple placeholder - you can add an icon file)
  const iconPath = join(__dirname, '../../assets/icon.png');
  
  // Create a simple icon if file doesn't exist
  let trayIcon;
  try {
    if (existsSync(iconPath)) {
      trayIcon = nativeImage.createFromPath(iconPath);
    } else {
      throw new Error('Icon file not found');
    }
  } catch (error) {
    // Create a simple template image for macOS (16x16 white square with D)
    // Template images are automatically colored by macOS
    const { createCanvas } = require('canvas');
    try {
      const canvas = createCanvas(16, 16);
      const ctx = canvas.getContext('2d');
      ctx.fillStyle = 'white';
      ctx.fillRect(0, 0, 16, 16);
      ctx.fillStyle = 'black';
      ctx.font = 'bold 12px Arial';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText('D', 8, 8);
      const buffer = canvas.toBuffer('image/png');
      trayIcon = nativeImage.createFromBuffer(buffer);
      trayIcon.setTemplateImage(true); // Make it a template image for macOS
    } catch (canvasError) {
      // Fallback: create a simple 16x16 image using nativeImage
      const simpleIcon = nativeImage.createEmpty();
      // On macOS, we can use a simple approach
      trayIcon = simpleIcon;
    }
    console.log('Using generated tray icon');
  }

  // Create tray with icon
  if (trayIcon && !trayIcon.isEmpty()) {
    tray = new Tray(trayIcon);
  } else {
    // Last resort: create a minimal visible icon
    const minimalIcon = nativeImage.createFromDataURL('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==');
    tray = new Tray(minimalIcon);
  }
  
  updateMenu();

  return {
    updateConnectionStatus: (status, message) => {
      connectionStatus = status;
      updateMenu(message);
    }
  };
};

function updateMenu(message) {
  const statusText = getStatusText(connectionStatus, message);
  
  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'DeskDrop',
      enabled: false
    },
    {
      type: 'separator'
    },
    {
      label: `Status: ${statusText}`,
      enabled: false
    },
    {
      type: 'separator'
    },
    {
      label: 'Reconnect',
      click: () => {
        if (reconnectCallback) {
          reconnectCallback();
          updateConnectionStatus('connecting', 'Reconnecting...');
        }
      },
      enabled: connectionStatus !== 'connected'
    },
    {
      type: 'separator'
    },
    {
      label: 'Quit',
      click: () => {
        app.quit();
      }
    }
  ]);

  tray.setContextMenu(contextMenu);
  tray.setToolTip(`DeskDrop - ${statusText}`);
}

function getStatusText(status, message) {
  switch (status) {
    case 'connected':
      return 'Connected';
    case 'connecting':
      return message || 'Connecting...';
    case 'disconnected':
      return message || 'Disconnected';
    case 'error':
      return message || 'Error';
    default:
      return 'Unknown';
  }
}

