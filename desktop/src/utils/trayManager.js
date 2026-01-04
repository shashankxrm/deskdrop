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
    // Fallback: use empty image (system will use default)
    trayIcon = undefined;
    console.log('Using default tray icon');
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

